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
	inputFlag := flag.String("input", "0", "opencv input address")
	flag.Parse()

	var window *gocv.Window

	if *headlessFlag == false {
		window = gocv.NewWindow("Motion Window")
		defer window.Close()
	}

	// webcam, err := gocv.OpenVideoCapture("udp://0.0.0.0:5000")
	// webcam, err := gocv.OpenVideoCapture("tcp://192.168.1.83:5000")
	webcam, err := gocv.OpenVideoCapture(*inputFlag)
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

	// imgScaled := gocv.NewMat()
	// defer imgScaled.Close()

	regionRect := image.Rect(29, 156, 859, 634)

	imgMask := gocv.IMRead("test_mask.png", gocv.IMReadGrayScale)
	if imgMask.Empty() {
		log.Fatalln("Mask not loaded")
	}
	gocv.Resize(imgMask, &imgMask, image.Point{width, height}, 0, 0, 1)
	imgMask = imgMask.Region(regionRect)
	// maskArea := maskArea(imgMask)

	detector := cam.NewDetector(imgMask)
	defer detector.Close()

	frameTime := time.Now()
	processTime := time.Now()

	fpsSum := 0
	processSum := 0
	framesCnt := 0

	imgResult := gocv.NewMat()
	defer imgResult.Close()

	for {
		if ok := webcam.Read(&imgRaw); !ok {
			log.Println("Reading frame from camera is not ok. Breaking")
			break
		}

		if imgRaw.Empty() {
			log.Println("Captured frame is empty")
			continue
		}

		// gocv.Resize(imgRaw, &imgScaled, image.Point{width, height}, 0, 0, 1)

		imgScaled := imgRaw.Region(regionRect)
		defer imgScaled.Close()

		processTime = time.Now()
		detector.Process(imgScaled, &imgResult)

		processDuration := time.Since(processTime)

		framesCnt++
		fpsSum += int(time.Second / time.Since(frameTime))
		processSum += int(processDuration / time.Millisecond)

		fmt.Printf("\rFPS: %d, process time: %d (current %s) ", fpsSum/framesCnt, processSum/framesCnt, processDuration)

		if window != nil {
			window.IMShow(imgRaw)

			if window.WaitKey(10) == 27 {
				break
			}
		}

		frameTime = time.Now()
	}
}
