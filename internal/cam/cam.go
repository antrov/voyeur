package cam

import (
	"image"
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

type commandType int

const (
	commandTypeClose commandType = iota
	commandTypeStartRecording
	commandTypeStopRecording
	commandTypeCancelRecording
	commandTypeTakePhoto
	commandTypeStartDetection
	commandTypeStopDetection
	// CaptureCommand
)

type captureCommand struct {
	cmdType commandType
}

type EventType int

const (
	// EventTypeCaptureStarted is for started capture of video
	EventTypeCaptureStarted EventType = iota + 1
	// EventTypeCaptureStopped is for stopped capture of video
	EventTypeCaptureStopped
	// EventTypePhotoAvailable is when single frame is captured
	EventTypePhotoAvailable
	// EventTypeRecordingAvailable is when recording file is ready to use
	EventTypeRecordingAvailable
	// EventTypeDetection is when any new detection in ROI
	EventTypeDetection
)

// CaptureEvent I am Channel
type CaptureEvent struct {
	Type EventType
	File string
}

type CamCapture struct {
	sourceID    int
	commandChan chan captureCommand
	EventsChan  chan CaptureEvent
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

func (c *CamCapture) TakePhoto() chan CaptureEvent {
	evtChan, cmdChan := c.startSessionIfNeeded()

	cmdChan <- captureCommand{commandTypeTakePhoto}

	return evtChan
}

func (c *CamCapture) StartRecording() chan CaptureEvent {
	evtChan, cmdChan := c.startSessionIfNeeded()

	cmdChan <- captureCommand{commandTypeStartRecording}

	return evtChan
}

func (c *CamCapture) StopRecording() {
	if c.commandChan == nil {
		return
	}

	c.commandChan <- captureCommand{commandTypeStopRecording}
}

func (c *CamCapture) CancelRecording() {
	if c.commandChan == nil {
		return
	}

	c.commandChan <- captureCommand{commandTypeCancelRecording}
}

func (c *CamCapture) StartDetection() chan CaptureEvent {
	evtChan, cmdChan := c.startSessionIfNeeded()

	cmdChan <- captureCommand{commandTypeStartDetection}

	return evtChan
}

func createCaptureFilename(ext string) string {
	return time.Now().Format("2006-01-02T15-04-05") + ext
}

func (c *CamCapture) startSessionIfNeeded() (chan CaptureEvent, chan captureCommand) {
	if c.commandChan != nil && c.EventsChan != nil {
		return c.EventsChan, c.commandChan
	}

	ca := make(chan CaptureEvent, 20)
	cmd := make(chan captureCommand, 20)

	c.commandChan = cmd
	c.EventsChan = ca

	go startSession(c.sourceID, ca, cmd)

	return ca, cmd
}

func startSession(s int, c chan CaptureEvent, cmd chan captureCommand) {
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

	// var detectionTime *time.Time
	// var detectionAlarmed bool
	// var writerFilename *string
	// var writer *gocv.VideoWriter

	takePhoto := false
	detectionEnabled := false

	defer func() {
		c <- CaptureEvent{Type: EventTypeCaptureStopped}
		// if writer != nil {
		// 	println("writer closed")
		// 	writer.Close()
		// }
	}()

	c <- CaptureEvent{Type: EventTypeCaptureStarted}
	for {
		select {
		case com := <-cmd:
			println("command", com.cmdType)
			switch com.cmdType {
			case commandTypeClose:

				return

			case commandTypeTakePhoto:
				takePhoto = true

			case commandTypeStartDetection:
				detectionEnabled = true

			case commandTypeStopDetection:
				detectionEnabled = false
			default:
			}
		default:
		}

		if !takePhoto && !detectionEnabled {
			break
		}

		if ok := webcam.Read(&imgRaw); !ok {
			break
		}

		if imgRaw.Empty() {
			continue
		}

		if takePhoto {
			println("take photo is true")
			takePhoto = false
			img := gocv.NewMat()
			imgRaw.CopyTo(&img)
			println("copied photo")
			go func(c chan CaptureEvent, img gocv.Mat) {
				file := createCaptureFilename(".png")
				gocv.IMWrite(file, img)
				println("photos saved")
				img.Close()
				println("photo closed")
				c <- CaptureEvent{
					Type: EventTypePhotoAvailable,
					File: file}
				println("event sent")
			}(c, img)
		}

		if detectionEnabled {

			gocv.Resize(imgRaw, &imgRaw, image.Pt(0, 0), 0.5, 0.5, 1)
			gocv.CvtColor(imgRaw, &img, gocv.ColorRGBToGray)
			gocv.BitwiseAnd(img, imgMask, &img)
			mog2.Apply(img, &imgDelta)
			gocv.Threshold(imgDelta, &imgThresh, 25, 255, gocv.ThresholdBinary)

			// then dilate
			kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
			defer kernel.Close()
			gocv.Dilate(imgThresh, &imgThresh, kernel)
		}

		// contours := gocv.FindContours(imgThresh, gocv.RetrievalExternal, gocv.ChainApproxSimple)

		// for _, contour := range contours {
		// 	area := gocv.ContourArea(contour)
		// 	if area < MinimumArea {
		// 		continue
		// 	}

		// 	// c <- CaptureEvent{Type: EventTypeDetection}
		// 	break
		// }

		// now find contours
		// found := detectionAlarmed

		// if !detectionAlarmed {
		// 	contours := gocv.FindContours(imgThresh, gocv.RetrievalExternal, gocv.ChainApproxSimple)

		// 	for i, c := range contours {
		// 		area := gocv.ContourArea(c)
		// 		if area < MinimumArea {
		// 			continue
		// 		}

		// 		found = true

		// 		gocv.DrawContours(&imgRaw, contours, i, color.RGBA{255, 0, 0, 0}, 2)

		// 		rect := gocv.BoundingRect(c)
		// 		gocv.Rectangle(&imgRaw, rect, color.RGBA{0, 0, 255, 0}, 2)
		// 	}
		// }

		// print(detectionConfirmed)

		// resetState := func(cancelled bool) {
		// 	detectionTime = nil
		// 	detectionAlarmed = false
		// 	if writer != nil {
		// 		writer.Close()
		// 		writer = nil
		// 	}

		// 	if writerFilename == nil {
		// 		return
		// 	}

		// 	if cancelled {
		// 		os.Remove(*writerFilename)
		// 		writerFilename = nil
		// 	} else {

		// 	}
		// }

		// switch {
		// case !found:
		// 	if detectionTime != nil && !detectionAlarmed && time.Since(*detectionTime) >= cancellationThreshold {
		// 		println("Cancelled")
		// 		c <- CaptureEvent{"Cancelled"}
		// 		resetState(true)
		// 	}

		// case detectionTime == nil:
		// 	println("First detection")
		// 	c <- CaptureEvent{"First detection"}
		// 	now := time.Now()
		// 	detectionTime = &now

		// case time.Since(*detectionTime) >= waitDuration:
		// 	println("Ready for new events")
		// 	c <- CaptureEvent{"Ready for new events"}
		// 	resetState(false)

		// case time.Since(*detectionTime) >= recordingDuration:
		// 	// if writer != nil {
		// 	// println("Recording done")
		// 	// c <- CaptureEvent{"Recording done"}
		// 	// writer.Close()
		// 	// writer = nil

		// 	// writerFilename = nil
		// 	// }

		// case time.Since(*detectionTime) >= alarmThreshold:
		// 	if !detectionAlarmed {
		// 		println("Alarm")
		// 		c <- CaptureEvent{"Alarm"}
		// 		detectionAlarmed = true
		// 	}

		// case time.Since(*detectionTime) >= detectionThreshold:
		// 	// if writer == nil {
		// 	println("Starting recording")
		// 	c <- CaptureEvent{"Starting recording"}
		// 	// 	name := time.Now().Format("2006-01-02T15-04-05") + ".avi"

		// 	// 	writerFilename = &name
		// 	// 	writer, _ = gocv.VideoWriterFile(name, "MJPG", fps, imgRaw.Size()[1], imgRaw.Size()[0], true)
		// 	// }
		// }

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
	if c.commandChan == nil {
		return
	}

	// c.commands <- CommandChannel{}
	close(c.commandChan)
	c.commandChan = nil
}

func NewMask(buf []byte) (string, error) {
	draw, err := gocv.IMDecode(buf, gocv.IMReadUnchanged)
	if err != nil {
		return "", err
	}
	defer draw.Close()

	mask := gocv.NewMat()
	defer mask.Close()

	preview := gocv.NewMat()
	defer preview.Close()

	createMask(draw, draw, &mask, &preview)

	file := createCaptureFilename(".png")

	gocv.IMWrite(file, preview)

	return file, nil
}
