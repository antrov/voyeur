package cam

import (
	"image"
	"image/color"
	"os"
	"time"

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

const ()

type CommandChannel struct {
}

// CaptureEvent I am Channel
type CaptureEvent struct {
	Message string
}

type CamCapture struct {
	sourceID int
	commands chan CommandChannel
	// Events   chan CamChannel
	// commands	*chan
}

func New(source int) (*CamCapture, error) {
	// window := gocv.NewWindow("Motion Window")
	// defer window.Close()

	capture := CamCapture{
		sourceID: source,
	}

	return &capture, nil
}

func (c *CamCapture) SetROI(file string) {
	//zapis do cmdCannel
}

// Start capture to file. If duration = 0, saves only one frame as png, otherwise mp4
func (c *CamCapture) Capture(duration time.Duration) {
	// jeśli jest sesja - zmień czas
	// jeśli nie ma sesji - czas + rozpocznij
}

func (c *CamCapture) StartDetection() chan CaptureEvent {
	// jeśli jest sesja - ustaw tylko flagę detekcji
	// jeśli nie ma sesji - flaga + rozpocznij
	ca := make(chan CaptureEvent)
	cmd := make(chan CommandChannel)

	c.commands = cmd

	go startSession(c.sourceID, ca, cmd)

	return ca
}

func startSession(s int, c chan CaptureEvent, cmd chan CommandChannel) {
	webcam, err := gocv.OpenVideoCapture(s)
	if err != nil {
		return
	}
	defer webcam.Close()

	// fps := webcam.Get(gocv.VideoCaptureFPS)

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
		c <- CaptureEvent{"Stopped"}
		if writer != nil {
			println("writer closed")
			writer.Close()
		}
	}()

	c <- CaptureEvent{"Starting detection"}
	for {
		select {
		case <-cmd:
			return
		default:
		}

		if ok := webcam.Read(&imgRaw); !ok {
			break
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
				c <- CaptureEvent{"Cancelled"}
				resetState(true)
			}

		case detectionTime == nil:
			println("First detection")
			c <- CaptureEvent{"First detection"}
			now := time.Now()
			detectionTime = &now

		case time.Since(*detectionTime) >= waitDuration:
			println("Ready for new events")
			c <- CaptureEvent{"Ready for new events"}
			resetState(false)

		case time.Since(*detectionTime) >= recordingDuration:
			// if writer != nil {
			println("Recording done")
			c <- CaptureEvent{"Recording done"}
			// writer.Close()
			// writer = nil

			// writerFilename = nil
			// }

		case time.Since(*detectionTime) >= alarmThreshold:
			if !detectionAlarmed {
				println("Alarm")
				c <- CaptureEvent{"Alarm"}
				detectionAlarmed = true
			}

		case time.Since(*detectionTime) >= detectionThreshold:
			// if writer == nil {
			println("Starting recording")
			c <- CaptureEvent{"Starting recording"}
			// 	name := time.Now().Format("2006-01-02T15-04-05") + ".avi"

			// 	writerFilename = &name
			// 	writer, _ = gocv.VideoWriterFile(name, "MJPG", fps, imgRaw.Size()[1], imgRaw.Size()[0], true)
			// }
		}

		// if writer != nil {
		// 	writer.Write(imgRaw)
		// }

		// window.IMShow(img)

		// if window.WaitKey(10) == 27 {
		// 	break
		// }

	}

}

func (c *CamCapture) Close() {
	if c.commands == nil {
		return
	}

	c.commands <- CommandChannel{}
}
