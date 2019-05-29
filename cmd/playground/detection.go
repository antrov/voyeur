package main

import (
	"flag"
	"fmt"
	"image"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"

	"gitlab.com/antrov/voyeur/internal/cam"
	"gocv.io/x/gocv"
)

var (
	headlessFlag = flag.Bool("headless", false, "show results in window")
	inputFlag    = flag.String("input", "0", "opencv input address/file/video index")
	cpuprofile   = flag.String("cpuprofile", "", "write cpu profile to `file`")
	width        = flag.Float64("width", 960, "width")
)

func main() {
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}

	}

	var window *gocv.Window

	if *headlessFlag == false {
		window = gocv.NewWindow("Motion Window")
		defer window.Close()
	}

	var gracefulStop = make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	go func() {
		sig := <-gracefulStop
		fmt.Printf("caught sig: %+v", sig)
		pprof.StopCPUProfile()
		os.Exit(0)
	}()

	webcam, err := gocv.OpenVideoCapture(*inputFlag)
	if err != nil {
		return
	}

	webcam.Set(gocv.VideoCaptureFrameWidth, *width)
	webcam.Set(gocv.VideoCaptureFrameHeight, float64(int(*width*0.75)))
	defer webcam.Close()

	w := int(webcam.Get(gocv.VideoCaptureFrameWidth))
	h := int(webcam.Get(gocv.VideoCaptureFrameHeight))
	s := image.Point{w, h}

	imgRaw := gocv.NewMat()
	defer imgRaw.Close()

	imgResult := gocv.NewMat()
	defer imgResult.Close()

	roi, err := cam.NewROI("resources/testing/test_mask.png", s)
	if err != nil {
		log.Fatal(err)
	}
	defer roi.Close()

	detectorDiff := cam.NewDetectorDiff()

	var detector cam.Detector = &detectorDiff
	defer detector.Close()

	detectorDiff2 := cam.NewDetectorDiff()

	var detector2 cam.Detector = &detectorDiff2
	defer detector2.Close()

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

		roi.Apply(imgRaw, &imgResult)

		processTime = time.Now()
		r := image.Rect(0, 0, roi.Bounds.Size().X/2, roi.Bounds.Size().Y)
		r2 := image.Rect(roi.Bounds.Size().X/2, 0, roi.Bounds.Size().X, roi.Bounds.Size().Y)

		p1 := imgResult.Region(r)
		p2 := imgResult.Region(r2)

		go detector.Process(p1, &p1)
		go detector2.Process(p2, &p2)

		<-detectorDiff.C
		<-detectorDiff2.C

		p1.Close()
		p2.Close()

		processDuration := time.Since(processTime)

		framesCnt++
		fpsSum += int(time.Second / time.Since(frameTime))
		processSum += int(processDuration / time.Millisecond)

		fmt.Printf("\rFPS: %d, process time: %d (current %s) ", fpsSum/framesCnt, processSum/framesCnt, processDuration)

		if window != nil {
			window.IMShow(imgResult)

			if window.WaitKey(10) == 27 {
				break
			}
		}

		frameTime = time.Now()
	}
}
