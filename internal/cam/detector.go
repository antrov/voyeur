package cam

import (
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

// Detector is responsible for processing and detecting movement in frame
type Detector struct {
	ImgDelta  gocv.Mat
	ImgThresh gocv.Mat
	ImgMask   gocv.Mat
	mog2      gocv.BackgroundSubtractorMOG2
	ImgBack   gocv.Mat
}

// UpdateMask sets new mask Mat which be used on next detection
func (d *Detector) UpdateMask(mask gocv.Mat) {
	d.ImgMask.Close()
	d.ImgMask = mask
}

// Close all gocv.Mats
func (d *Detector) Close() {
	d.ImgDelta.Close()
	d.ImgThresh.Close()
	d.ImgMask.Close()
	d.mog2.Close()
}

// Clear state of cross-images
func (d *Detector) Clear() {
	// zeros := gocv.NewMat()
	// zeros.CopyTo(&img)
	// zeros.CopyTo(&imgDelta)
	// zeros.CopyTo(&imgThresh)
}

// NewDetector creates new instance of detector with all Mats initialized
func NewDetector(mask gocv.Mat) Detector {
	return Detector{
		ImgDelta:  gocv.NewMat(),
		ImgThresh: gocv.NewMat(),
		ImgMask:   mask,
		mog2:      gocv.NewBackgroundSubtractorMOG2(),
	}
}

// Process for procesing image
func (d *Detector) Process(imgRaw gocv.Mat, imgDst *gocv.Mat) (bool, []gocv.Mat) {
	if &d.ImgBack == nil {
		d.ImgBack = imgRaw
	}

	found := false
	stages := []gocv.Mat{}

	stages = append(stages, imgRaw)

	img := gocv.NewMat()
	defer img.Close()

	appendImg := func(i gocv.Mat) {
		imgCopy := gocv.NewMat()
		i.CopyTo(&imgCopy)
		stages = append(stages, imgCopy)
	}

	// wyrównywanie jasności
	// gocv.CvtColor(imgRaw, &img, gocv.ColorBGRToYUV)
	// channels := gocv.Split(img)
	// gocv.EqualizeHist(channels[0], &channels[0])
	// gocv.Merge(channels, &img)
	// gocv.CvtColor(img, &img, gocv.ColorYUVToBGR)

	// imgEqual := gocv.NewMat()
	// img.CopyTo(&imgEqual)
	// stages = append(stages, imgEqual)

	gocv.GaussianBlur(imgRaw, &img, image.Point{3, 3}, 9, 9, gocv.BorderDefault)

	// gocv.BitwiseAnd(img, d.ImgMask, &img)
	d.mog2.Apply(img, &d.ImgDelta)

	bgr := gocv.NewMat()
	gocv.CvtColor(d.ImgDelta, &bgr, gocv.ColorGrayToBGR)
	appendImg(bgr)

	gocv.Threshold(d.ImgDelta, &d.ImgThresh, 25, 255, gocv.ThresholdBinary)

	bgr = gocv.NewMat()
	gocv.CvtColor(d.ImgThresh, &bgr, gocv.ColorGrayToBGR)
	appendImg(bgr)

	// then dilate
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
	defer kernel.Close()
	gocv.Dilate(d.ImgThresh, &d.ImgThresh, kernel)

	bgr = gocv.NewMat()
	gocv.CvtColor(d.ImgThresh, &bgr, gocv.ColorGrayToBGR)
	appendImg(bgr)

	contours := gocv.FindContours(d.ImgThresh, gocv.RetrievalExternal, gocv.ChainApproxSimple)
	drawings := gocv.NewMat()
	imgRaw.CopyTo(&drawings)

	for i, contour := range contours {
		area := gocv.ContourArea(contour) // / maskArea * 100
		// // log.Println(area)
		// if area <= detectionMinArea || area >= detectionMaxArea {
		// 	continue
		// }

		if area <= 100 {
			continue
		}

		found = true
		gocv.DrawContours(&drawings, contours, i, color.RGBA{255, 0, 0, 0}, 2)
	}

	appendImg(drawings)

	return found, stages
}
