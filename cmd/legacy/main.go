package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"time"

	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	"gocv.io/x/gocv"
)

const MinimumArea = 300

const (
	cancellationThreshold = time.Duration(500 * time.Millisecond)
	detectionThreshold    = time.Duration(100 * time.Millisecond)
	alarmThreshold        = time.Duration(1 * time.Second)
	waitDuration          = time.Duration(1 * time.Minute)
	recordingDuration     = time.Duration(5 * time.Second)
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	rand.Seed(time.Now().Unix())

	if len(os.Args) < 2 {
		fmt.Println("How to run:\n\tmotion-detect [camera ID]")
		return
	}

	window := gocv.NewWindow("Motion Window")
	defer window.Close()

	// parse args
	deviceID := os.Args[1]

	webcam, err := gocv.OpenVideoCapture(deviceID)
	if err != nil {
		fmt.Printf("Error opening video capture device: %v\n", deviceID)
		return
	}
	defer webcam.Close()

	fps := webcam.Get(gocv.VideoCaptureFPS)

	imgRaw := gocv.NewMat()
	defer imgRaw.Close()

	img := gocv.NewMat()
	defer img.Close()

	imgDelta := gocv.NewMat()
	defer imgDelta.Close()

	imgThresh := gocv.NewMat()
	defer imgThresh.Close()

	imgMask := gocv.IMRead("mask.png", gocv.IMReadGrayScale)
	gocv.Resize(imgMask, &imgMask, image.Pt(0, 0), 0.5, 0.5, 1)
	defer imgMask.Close()

	mog2 := gocv.NewBackgroundSubtractorMOG2()
	defer mog2.Close()

	var detectionTime *time.Time
	var detectionAlarmed bool
	var writerFilename *string
	var writer *gocv.VideoWriter

	defer func() {
		if writer != nil {
			println("writer closed")
			writer.Close()
		}
	}()

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	// u := tgbotapi.NewUpdate(0)
	// u.Timeout = 60

	// updates, _ := bot.GetUpdatesChan(u)

	fmt.Printf("Start reading device: %v\n", deviceID)
	for {
		if ok := webcam.Read(&imgRaw); !ok {
			fmt.Printf("Device closed: %v\n", deviceID)
			return
		}
		if imgRaw.Empty() {
			continue
		}

		gocv.Resize(imgRaw, &imgRaw, image.Pt(0, 0), 0.5, 0.5, 1)
		gocv.CvtColor(imgRaw, &img, gocv.ColorRGBToGray)
		gocv.BitwiseAnd(img, imgMask, &img)
		mog2.Apply(img, &imgDelta)
		gocv.Threshold(imgDelta, &imgThresh, 25, 255, gocv.ThresholdBinary)

		// then dilate
		kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
		defer kernel.Close()
		gocv.Dilate(imgThresh, &imgThresh, kernel)

		// now find contours
		found := detectionAlarmed

		if !detectionAlarmed {
			contours := gocv.FindContours(imgThresh, gocv.RetrievalExternal, gocv.ChainApproxSimple)

			for i, c := range contours {
				area := gocv.ContourArea(c)
				if area < MinimumArea {
					continue
				}

				found = true

				gocv.DrawContours(&imgRaw, contours, i, color.RGBA{255, 0, 0, 0}, 2)

				rect := gocv.BoundingRect(c)
				gocv.Rectangle(&imgRaw, rect, color.RGBA{0, 0, 255, 0}, 2)
			}
		}

		// print(detectionConfirmed)

		resetState := func(cancelled bool) {
			detectionTime = nil
			detectionAlarmed = false
			if writer != nil {
				writer.Close()
				writer = nil
			}

			if writerFilename == nil {
				return
			}

			if cancelled {
				os.Remove(*writerFilename)
				writerFilename = nil
			} else {

			}
		}

		switch {
		case !found:
			if detectionTime != nil && !detectionAlarmed && time.Since(*detectionTime) >= cancellationThreshold {
				println("Cancelled")
				resetState(true)
			}

		case detectionTime == nil:
			println("First detection")
			now := time.Now()
			detectionTime = &now

		case time.Since(*detectionTime) >= waitDuration:
			println("Ready for new events")
			resetState(false)

		case time.Since(*detectionTime) >= recordingDuration:
			if writer != nil {
				println("Recording done")
				writer.Close()
				writer = nil

				name := *writerFilename
				writerFilename = nil

				go func() {
					println("Exporting")
					cmd := exec.Command("ffmpeg", "-i", name, "-vcodec", "libx264", "-f", "mp4", "-y", name+".mp4")
					var stderr bytes.Buffer
					cmd.Stderr = &stderr
					err := cmd.Run()
					if err != nil {
						println("cmd error")
						fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
						return
					}
					println("exported")
					msg := tgbotapi.NewVideoUpload(411400390, name+".mp4")
					_, err = bot.Send(msg)
					if err != nil {
						println(err)
					}

				}()
			}

		case time.Since(*detectionTime) >= alarmThreshold:
			if !detectionAlarmed {
				println("Alarm")
				detectionAlarmed = true
				files, err := ioutil.ReadDir("./sounds")
				f, err := os.Open("./sounds/" + files[rand.Intn(len(files))].Name())
				if err != nil {
					log.Fatal(err)
				}

				streamer, format, err := mp3.Decode(f)
				if err != nil {
					log.Fatal(err)
				}
				defer streamer.Close()

				speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
				speaker.Play(streamer)
			}

		case time.Since(*detectionTime) >= detectionThreshold:
			if writer == nil {
				println("Starting recording")
				name := time.Now().Format("2006-01-02T15-04-05") + ".avi"

				writerFilename = &name
				writer, _ = gocv.VideoWriterFile(name, "MJPG", fps, imgRaw.Size()[1], imgRaw.Size()[0], true)
			}
		}

		if writer != nil {
			writer.Write(imgRaw)
		}

		window.IMShow(img)

		if window.WaitKey(10) == 27 {
			break
		}
	}
}
