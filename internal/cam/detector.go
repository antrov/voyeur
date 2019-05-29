package cam

import (
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

// EM is empty Mat type (zero filled)
var EM = gocv.NewMat()

// Detector contains processing sequence with resulting bollean value
type Detector interface {
	Process(src gocv.Mat, dst *gocv.Mat) bool
	Close()
	Reset()
}

// DetectorKNN uses BackgroundSubtractorKNN under the hood
// Based on https://github.com/hybridgroup/gocv/blob/master/cmd/motion-detect/main.go
//
// BackgroundSubtractorKNN is quite CPU expensive
// DetectorKNN is sensitive for light/white balance changes
type DetectorKNN struct {
	delta  gocv.Mat
	thresh gocv.Mat
	mog2   gocv.BackgroundSubtractorKNN
}

// NewDetectorKNN creates new instance of detector with all Mats initialized
func NewDetectorKNN() DetectorKNN {
	return DetectorKNN{
		delta:  gocv.NewMat(),
		thresh: gocv.NewMat(),
		mog2:   gocv.NewBackgroundSubtractorKNN(),
	}
}

// Process for procesing image
func (d *DetectorKNN) Process(src gocv.Mat, dst *gocv.Mat) bool {
	found := false

	// imgRaw.CopyTo(imgDst)

	d.mog2.Apply(src, &d.delta)

	// remaining cleanup of the image to use for finding contours.
	// first use threshold
	gocv.Threshold(d.delta, &d.thresh, 25, 255, gocv.ThresholdBinary)

	// then dilate
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
	defer kernel.Close()
	gocv.Dilate(d.thresh, &d.thresh, kernel)

	// now find contours
	gocv.CvtColor(d.thresh, dst, gocv.ColorGrayToBGR)

	contours := gocv.FindContours(d.thresh, gocv.RetrievalExternal, gocv.ChainApproxSimple)
	for i, c := range contours {
		area := gocv.ContourArea(c)
		if area < 50 {
			continue
		}

		found = true
		// break

		// status = "Motion detected"
		statusColor := color.RGBA{255, 0, 0, 0}
		gocv.DrawContours(dst, contours, i, statusColor, 2)

		// rect := gocv.BoundingRect(c)
		// gocv.Rectangle(imgDst, rect, color.RGBA{0, 0, 255, 0}, 2)
		// gocv.PutText(imgDst, fmt.Sprintf("%.2f", area), rect.Min, gocv.FontHersheyPlain, 1.2, statusColor, 2)
		// break
	}

	return found
}

// Close all gocv.Mats
func (d *DetectorKNN) Close() {
	d.delta.Close()
	d.thresh.Close()
	d.mog2.Close()
}

// Clear state of cross-images
func (d *DetectorKNN) Reset() {
	EM.CopyTo(&d.delta)
	EM.CopyTo(&d.thresh)
}

// DetectorDiff uses running average and diff for detect changes between frames
// Bases on http://www.robindavid.fr/opencv-tutorial/motion-detection-with-opencv.html
//
// Is twice more efficient than KNN, is no sensitive for changes in light
// Sensitiveness for changes can be controlled by thresholdlevel and average alpha blending
type DetectorDiff struct {
	grayImg   gocv.Mat
	movingAvg gocv.Mat
	diff      gocv.Mat
	temp      gocv.Mat
}

// NewDetectorDiff creates new instance DetectorDiff
func NewDetectorDiff() DetectorDiff {
	return DetectorDiff{
		grayImg:   gocv.NewMat(),
		movingAvg: gocv.NewMat(),
		diff:      gocv.NewMat(),
		temp:      gocv.NewMat(),
	}
}

// Process for procesing image
func (d *DetectorDiff) Process(src gocv.Mat, dst *gocv.Mat) bool {
	gocv.Blur(src, &src, image.Point{3, 3})

	if d.diff.Empty() {
		src.CopyTo(&d.diff)
		src.CopyTo(&d.temp)
		gocv.ConvertScaleAbs(src, &d.movingAvg, 1, 1)
	} else {
		// Speed of change - higher value - higher sensitivity for small changes
		a := 0.2
		gocv.AddWeighted(src, 1-a, d.movingAvg, a, 0, &d.movingAvg)
	}

	gocv.ConvertScaleAbs(d.movingAvg, &d.temp, 1, 0)
	gocv.AbsDiff(src, d.temp, &d.diff)

	gocv.CvtColor(d.diff, &d.grayImg, gocv.ColorBGRToGray)
	gocv.Threshold(d.grayImg, &d.grayImg, 10, 255, gocv.ThresholdBinary)

	gocv.Dilate(d.grayImg, &d.grayImg, EM)
	gocv.Erode(d.grayImg, &d.grayImg, EM)

	d.grayImg.CopyTo(dst)

	// contours := gocv.FindContours(d.grayImg, gocv.RetrievalExternal, gocv.ChainApproxSimple)
	// for i, c := range contours {
	// 	area := gocv.ContourArea(c)
	// 	if area < 50 {
	// 		continue
	// 	}

	// 	statusColor := color.RGBA{255, 0, 0, 0}
	// 	// gocv.DrawContours(imgDst, contours, i, statusColor, 2)
	// }

	return false
}

// Close all gocv.Mats
func (d *DetectorDiff) Close() {
	d.diff.Close()
	d.grayImg.Close()
	d.movingAvg.Close()
	d.temp.Close()
}

// Reset state of cross-images
func (d *DetectorDiff) Reset() {
	EM.CopyTo(&d.diff)
	EM.CopyTo(&d.grayImg)
	EM.CopyTo(&d.movingAvg)
	EM.CopyTo(&d.temp)
}
