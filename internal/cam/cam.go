package cam

import (
	"image"
	"image/color"
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
func StartSession(sourceID int, maskf string, evtChan chan CaptureEvent, cmdChan chan CaptureCommandType, window *gocv.Window) {
	webcam, err := gocv.OpenVideoCapture(sourceID)
	if err != nil {
		return
	}
	defer webcam.Close()

	ratio := webcam.Get(gocv.VideoCaptureFrameHeight) / webcam.Get(gocv.VideoCaptureFrameWidth)
	width := 640
	height := int(float64(width) * ratio) //int(webcam.Get(gocv.VideoCaptureFrameHeight))

	imgRaw := gocv.NewMat()
	defer imgRaw.Close()

	img := gocv.NewMat()
	defer img.Close()

	imgDelta := gocv.NewMat()
	defer imgDelta.Close()

	imgThresh := gocv.NewMat()
	defer imgThresh.Close()

	imgMask := gocv.IMRead(maskf, gocv.IMReadGrayScale)
	if imgMask.Empty() {
		log.Fatalln("Mask not loaded")
	}
	gocv.Resize(imgMask, &imgMask, image.Point{width, height}, 0, 0, 1)
	defer imgMask.Close()

	maskArea := maskArea(imgMask)

	mog2 := gocv.NewBackgroundSubtractorMOG2()
	defer mog2.Close()

	// recording vars
	var writerFilename *string
	var writer *gocv.VideoWriter

	// flags for actions
	takePhoto := false
	detectionEnabled := false
	previewROI := false

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
				detectionEnabled = true

			case CaptureCommandTypeStopDetection:
				detectionEnabled = false

				zeros := gocv.NewMat()
				zeros.CopyTo(&img)
				zeros.CopyTo(&imgDelta)
				zeros.CopyTo(&imgThresh)

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

		gocv.Resize(imgRaw, &imgRaw, image.Point{width, height}, 0, 0, 1)

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

			evtChan <- CaptureEvent{
				Type: EventTypePhotoAvailable,
				File: file}
		}

		if takePhoto {
			takePhoto = false
			file := createCaptureFilename(".png")
			gocv.IMWrite(file, imgRaw)

			evtChan <- CaptureEvent{
				Type: EventTypePhotoAvailable,
				File: file}
		}

		// concat := gocv.NewMat()
		// defer concat.Close()

		if detectionEnabled {
			// gocv.CvtColor(imgRaw, &concat, gocv.ColorBGRToGray)

			gocv.CvtColor(imgRaw, &img, gocv.ColorBGRToYUV)
			channels := gocv.Split(img)
			gocv.EqualizeHist(channels[0], &channels[0])
			gocv.Merge(channels, &img)
			gocv.CvtColor(img, &img, gocv.ColorYUVToBGR)
			gocv.GaussianBlur(img, &img, image.Point{3, 3}, 9, 9, gocv.BorderDefault)

			gocv.CvtColor(img, &img, gocv.ColorRGBToGray)
			gocv.BitwiseAnd(img, imgMask, &img)
			mog2.Apply(img, &imgDelta)
			gocv.Threshold(imgDelta, &imgThresh, 25, 255, gocv.ThresholdBinary)

			// then dilate
			kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
			defer kernel.Close()
			gocv.Dilate(imgThresh, &imgThresh, kernel)

			contours := gocv.FindContours(imgThresh, gocv.RetrievalExternal, gocv.ChainApproxSimple)

			for i, contour := range contours {
				area := gocv.ContourArea(contour) / maskArea * 100
				// log.Println(area)
				if area <= detectionMinArea || area >= detectionMaxArea {
					continue
				}

				gocv.DrawContours(&img, contours, i, color.RGBA{255, 0, 0, 0}, 2)
				evtChan <- CaptureEvent{Type: EventTypeDetection}
				break
			}

			// gocv.Hconcat(concat, img, &concat)
			// gocv.Hconcat(concat, imgDelta, &concat)
		}

		if writer != nil && writer.IsOpened() {
			writer.Write(imgRaw)
		}

		// window.IMShow(concat)

		// if window.WaitKey(10) == 27 {
		// 	break
		// }

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

	createMask(draw, draw, &mask, &preview)

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
