package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"log"
	"strconv"
	"time"

	"gocv.io/x/gocv"
)

var (
	width    = flag.Float64("width", 960, "width")
	duration = flag.Int64("duration", 2, "duration of recording. Set 0 to make it infinite")
	file     = flag.String("file", "capture.mkv", "path to output capture file")
	codec    = flag.String("codec", "h264", "Codec used to encode video (h264, png, jpeg)")
)

func main() {

	flag.Parse()
	height := int(*width * 0.75)

	webcam, err := gocv.OpenVideoCapture(0)
	if err != nil {
		return
	}
	defer webcam.Close()
	webcam.Set(gocv.VideoCaptureFrameWidth, *width)
	webcam.Set(gocv.VideoCaptureFrameHeight, float64(height))

	imgRaw := gocv.NewMat()
	defer imgRaw.Close()

	w := int(webcam.Get(gocv.VideoCaptureFrameWidth))
	h := int(webcam.Get(gocv.VideoCaptureFrameHeight))
	f := webcam.Get(gocv.VideoCaptureFPS)

	fmt.Printf("size is: %d x %d, FPS %f\n\r", w, h, f)

	bufLen := int(f) * int(*duration)
	buffer := make([]gocv.Mat, bufLen)

	frameTime := time.Now()
	frameIdx := 0

	frameColor := color.RGBA{255, 0, 0, 0}
	framePos := image.Point{30, 30}
	frameSize := float64(1.2)

	stopLoop := false
	var tickerChan <-chan time.Time

	if *duration > 0 {
		tickeer := time.NewTicker(time.Duration(*duration) * time.Second)
		tickerChan = tickeer.C
	}

	if webcam.Read(&imgRaw) {
		gocv.IMWrite("capture.png", imgRaw)
	}

	for {
		select {
		case <-tickerChan:
			stopLoop = true
		default:
		}

		if stopLoop {
			break
		}

		if ok := webcam.Read(&imgRaw); !ok {
			log.Println("Reading frame from camera is not ok. Breaking")
			break
		}

		if imgRaw.Empty() {
			log.Println("Captured frame is empty")
			continue
		}

		gocv.PutText(&imgRaw, strconv.Itoa(frameIdx), framePos, gocv.FontHersheyPlain, frameSize, frameColor, 3)

		if frameIdx < bufLen {
			buffer[frameIdx] = imgRaw.Clone()
		}

		fmt.Printf("\rFPS: %d\tframe %d\t", int(time.Second/time.Since(frameTime)), frameIdx)

		frameIdx++
		frameTime = time.Now()
	}

	writer, _ := gocv.VideoWriterFile(*file, *codec, f, w, h, true)

	for i, img := range buffer {
		if i >= frameIdx {
			break
		}

		writer.Write(img)
	}

	writer.Close()

}
