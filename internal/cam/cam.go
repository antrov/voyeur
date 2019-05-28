package cam

import (
	"image"
	"log"
	"os"
	"time"

	"gocv.io/x/gocv"
)

const (
	detectionMinArea = 10
	detectionMaxArea = 50
)

// CaptureCommandType to set capture state
type CaptureCommandType int

const (
	// CaptureCommandTypeClose CaptureCommandType = iota
	CaptureCommandTypeStartRecording CaptureCommandType = iota + 1
	CaptureCommandTypeStopRecording
	CaptureCommandTypeCancelRecording
	CaptureCommandTypeTakePhoto
	CaptureCommandTypeStartDetection
	CaptureCommandTypeStopDetection
	CaptureCommandTypePreviewROI
	// CaptureCommand
)

// EventType show what happens in capture loop
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

func createCaptureFilename(ext string) string {
	return "./captures/" + time.Now().Format("2006-01-02T15-04-05") + ext
}

// StartSession start capture loop. If no commands send, loop is idle
func StartSession(source interface{}, maskf string, evtChan chan CaptureEvent, cmdChan chan CaptureCommandType) {
	webcam, err := gocv.OpenVideoCapture(source)
	if err != nil {
		return
	}
	webcam.Set(gocv.VideoCaptureFrameWidth, 960)
	webcam.Set(gocv.VideoCaptureFrameHeight, 720)
	defer webcam.Close()

	width := int(webcam.Get(gocv.VideoCaptureFrameWidth))
	height := int(webcam.Get(gocv.VideoCaptureFrameHeight))

	imgRaw := gocv.NewMat()
	defer imgRaw.Close()

	// regionRect := image.Rect(29, 156, 859, 634)
	regionRect := image.Rect(20, 20, 400, 400)

	imgMask := gocv.IMRead(maskf, gocv.IMReadGrayScale)
	if imgMask.Empty() {
		log.Fatalln("Mask not loaded")
	}
	imgMask = imgMask.Region(regionRect)

	detector := NewDetector()
	defer detector.Close()

	// recording vars
	var writerFilename *string
	var writer *gocv.VideoWriter

	// flags for actions
	takePhoto := false
	detectionEnabled := false
	previewROI := false

	// telemetry and stats vars
	frameTime := time.Now()
	processTime := time.Now()

	fpsSum := 0
	processSum := 0
	framesCnt := 0

	defer func() {
		evtChan <- CaptureEvent{Type: EventTypeCaptureStopped}

		if writer != nil {
			writer.Close()
		}
	}()

	log.Println("Starting camera loop")

	evtChan <- CaptureEvent{Type: EventTypeCaptureStarted}
	for {
		select {
		case cmd := <-cmdChan:
			// println("command", cmd)

			switch cmd {
			// case CaptureCommandTypeClose:
			// 	return
			case CaptureCommandTypeTakePhoto:
				takePhoto = true

			case CaptureCommandTypeStartDetection:
				log.Println("Start detection")
				detectionEnabled = true

			case CaptureCommandTypeStopDetection:
				log.Println("Stop detection")
				detectionEnabled = false
				detector.Clear()

			case CaptureCommandTypePreviewROI:
				previewROI = true

			case CaptureCommandTypeStartRecording:
				if writer == nil {
					println("Starting recording")
					fps := webcam.Get(gocv.VideoCaptureFPS)
					name := createCaptureFilename(".avi")

					writerFilename = &name
					writer, _ = gocv.VideoWriterFile(name, "MJPG", fps, int(width), int(height), true)
				}

			case CaptureCommandTypeStopRecording:
				if writer != nil {

					println("Stop recording")
					writer.Close()

					evtChan <- CaptureEvent{
						Type: EventTypeRecordingAvailable,
						File: *writerFilename,
					}

					writer = nil
					writerFilename = nil
				}

			case CaptureCommandTypeCancelRecording:
				if writer != nil {

					println("Cancell recording")
					writer.Close()
					os.Remove(*writerFilename)

					writer = nil
					writerFilename = nil
				}

			default:
			}
		default:
		}

		if !takePhoto && !detectionEnabled && !previewROI {
			time.Sleep(time.Millisecond * 100)
			continue
		}

		if ok := webcam.Read(&imgRaw); !ok {
			log.Println("Reading frame from camera is not ok. Breaking")
			break
		}

		if imgRaw.Empty() {
			log.Println("Captured frame is empty")
			continue
		}

		imgScaled := imgRaw.Region(regionRect)
		defer imgScaled.Close()

		if takePhoto {
			takePhoto = false
			file := createCaptureFilename(".png")
			gocv.IMWrite(file, imgRaw)

			evtChan <- CaptureEvent{
				Type: EventTypePhotoAvailable,
				File: file}
		}

		if previewROI {
			previewROI = false

			file := createCaptureFilename(".png")
			imgMask = gocv.IMRead(maskf, gocv.IMReadGrayScale)

			previewMask := gocv.NewMat()
			defer previewMask.Close()

			preview := gocv.NewMat()
			defer preview.Close()

			previewROI := gocv.NewMat()
			defer previewROI.Close()

			gocv.Resize(imgMask, &imgMask, image.Point{width, height}, 0, 0, 1)
			gocv.CvtColor(imgMask, &previewMask, gocv.ColorGrayToBGR)
			gocv.BitwiseAnd(imgRaw, previewMask, &previewROI)
			gocv.AddWeighted(imgRaw, 0.3, previewROI, 1, 1, &preview)
			gocv.IMWrite(file, preview)

			imgMask = imgMask.Region(regionRect)
			log.Println("mask created, sending event")
			evtChan <- CaptureEvent{
				Type: EventTypePhotoAvailable,
				File: file}
		}

		if detectionEnabled {
			processTime = time.Now()
			if detector.Process(imgScaled, nil) {
				evtChan <- CaptureEvent{Type: EventTypeDetection}
			}

			processDuration := time.Since(processTime)

			framesCnt++
			fpsSum += int(time.Second / time.Since(frameTime))
			processSum += int(processDuration / time.Millisecond)

			// fmt.Printf("\rFPS: %d, process time: %d (current %s) ", fpsSum/framesCnt, processSum/framesCnt, processDuration)
		}

		if writer != nil && writer.IsOpened() {
			writer.Write(imgRaw)
		}

		frameTime = time.Now()
	}

}

// NewMask creates new roi from given buffer with red areas
func NewMask(buf []byte, file string) error {
	draw, err := gocv.IMDecode(buf, gocv.IMReadUnchanged)
	if err != nil {
		return err
	}
	defer draw.Close()

	mask := gocv.NewMat()
	defer mask.Close()

	preview := gocv.NewMat()
	defer preview.Close()

	// createMask(draw, draw, &mask, &preview)

	gocv.IMWrite(file, mask)

	return nil
}

func maskArea(imgMask gocv.Mat) float64 {
	contours := gocv.FindContours(imgMask, gocv.RetrievalExternal, gocv.ChainApproxSimple)
	totalArea := float64(0)

	for _, contour := range contours {
		area := gocv.ContourArea(contour)

		totalArea += area
	}

	return totalArea
}
