package cam

import (
	"fmt"
	"image/color"

	"gocv.io/x/gocv"
)

// Detector is responsible for processing and detecting movement in frame
type Detector struct {
	ImgDelta  gocv.Mat
	ImgThresh gocv.Mat
	ImgMask   gocv.Mat
	mog2      gocv.BackgroundSubtractorKNN
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
		mog2:      gocv.NewBackgroundSubtractorKNN(),
		ImgBack:   gocv.NewMat(),
	}
}

// Process for procesing image
func (d *Detector) Process(imgRaw gocv.Mat, imgDst *gocv.Mat) (bool, []gocv.Mat) {
	var stages []gocv.Mat

	found := false

	// imgRaw.CopyTo(imgDst)

	d.mog2.Apply(imgRaw, &d.ImgDelta)

	// remaining cleanup of the image to use for finding contours.
	// first use threshold
	gocv.Threshold(d.ImgDelta, &d.ImgThresh, 25, 255, gocv.ThresholdBinary)

	// then dilate
	// kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
	// defer kernel.Close()
	// gocv.Dilate(d.ImgThresh, &d.ImgThresh, kernel)

	d.ImgDelta.CopyTo(imgDst)
	gocv.CvtColor(*imgDst, imgDst, gocv.ColorGrayToBGR)

	// now find contours
	contours := gocv.FindContours(d.ImgThresh, gocv.RetrievalExternal, gocv.ChainApproxSimple)
	for i, c := range contours {
		area := gocv.ContourArea(c)
		if area < 50 {
			continue
		}

		found = true

		// status = "Motion detected"
		statusColor := color.RGBA{255, 0, 0, 0}
		gocv.DrawContours(imgDst, contours, i, statusColor, 2)

		rect := gocv.BoundingRect(c)
		gocv.Rectangle(imgDst, rect, color.RGBA{0, 0, 255, 0}, 2)
		gocv.PutText(imgDst, fmt.Sprintf("%.2f", area), rect.Min, gocv.FontHersheyPlain, 1.2, statusColor, 2)
		// break
	}

	return found, stages
}
