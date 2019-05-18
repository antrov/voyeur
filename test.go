package main

import (
	"flag"
	"fmt"
	"image"
	"log"
	"time"

	"gitlab.com/antrov/couch-watch/internal/cam"
	"gocv.io/x/gocv"
)

func main() {
	headlessFlag := flag.Bool("headless", false, "show results in window")
	flag.Parse()

	var window *gocv.Window

	if *headlessFlag == false {
		window = gocv.NewWindow("Motion Window")
		defer window.Close()
	}

	webcam, err := gocv.OpenVideoCapture(0)
	if err != nil {
		return
	}
	defer webcam.Close()

	ratio := webcam.Get(gocv.VideoCaptureFrameHeight) / webcam.Get(gocv.VideoCaptureFrameWidth)
	width := 640
	height := int(float64(width) * ratio) //int(webcam.Get(gocv.VideoCaptureFrameHeight))

	imgRaw := gocv.NewMat()
	defer imgRaw.Close()

	imgScaled := gocv.NewMat()
	defer imgScaled.Close()

	imgMask := gocv.IMRead("test_mask.png", gocv.IMReadGrayScale)
	if imgMask.Empty() {
		log.Fatalln("Mask not loaded")
	}
	gocv.Resize(imgMask, &imgMask, image.Point{width, height}, 0, 0, 1)
	// maskArea := maskArea(imgMask)

	detector := cam.NewDetector(imgMask)
	defer detector.Close()

	frameTime := time.Now()
	processTime := time.Now()

	fpsSum := 0
	processSum := 0
	framesCnt := 0

	for {
		if ok := webcam.Read(&imgRaw); !ok {
			log.Println("Reading frame from camera is not ok. Breaking")
			break
		}

		if imgRaw.Empty() {
			log.Println("Captured frame is empty")
			continue
		}

		gocv.Resize(imgRaw, &imgScaled, image.Point{width, height}, 0, 0, 1)

		processTime = time.Now()
		_, stages := detector.Process(imgScaled, nil)

		if window != nil {
			img := stages[0]

			for i, stage := range stages {
				if i > 0 {
					gocv.Hconcat(img, stage, &img)
				}
			}

			window.IMShow(img)

			if window.WaitKey(10) == 27 {
				break
			}
		}

		processDuration := time.Since(processTime)

		framesCnt++
		fpsSum += int(time.Second / time.Since(frameTime))
		processSum += int(processDuration / time.Millisecond)

		fmt.Printf("\rFPS: %d, process time: %d (current %s) ", fpsSum/framesCnt, processSum/framesCnt, processDuration)

		frameTime = time.Now()

		defer func() {
			for _, stage := range stages {
				stage.Close()
			}
		}()
	}
}
