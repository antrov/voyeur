package cam

import (
	"errors"
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

// ROI manages Region of Interest, allows to load mask from file
// or to create it from red drawings
type ROI struct {
	Mask       gocv.Mat
	Bounds     image.Rectangle
	contours   [][]image.Point
	maskRegion gocv.Mat
}

// ER is shorthand for empty ROI
var ER ROI

// NewROI from given file name or []byte
func NewROI(file string) (ROI, error) {
	maskMat := gocv.IMRead(file, gocv.IMReadGrayScale)

	if maskMat.Empty() {
		return ER, errors.New("Loaded mask is empty")
	}

	contours, bounds := contoursAndBounds(maskMat)
	if len(contours) == 0 {
		return ER, errors.New("Mask invalid - has no white ares")
	}

	gocv.CvtColor(maskMat, &maskMat, gocv.ColorGrayToBGR)

	roi := ROI{
		Mask:       maskMat,
		Bounds:     bounds,
		contours:   contours,
		maskRegion: maskMat.Region(bounds),
	}

	return roi, nil
}

// NewROIFromDrawing decodes bytes and processing
// red drawing to find and create mask
func NewROIFromDrawing(src []byte) (ROI, error) {
	maskSrc, err := gocv.IMDecode(src, gocv.IMReadUnchanged)
	if err != nil {
		return ER, err
	}
	defer maskSrc.Close()

	width := maskSrc.Size()[0]
	height := maskSrc.Size()[1]
	mask := gocv.NewMatWithSize(width, height, gocv.MatTypeCV8UC3)

	hsv := gocv.NewMat()
	defer hsv.Close()

	gocv.CvtColor(maskSrc, &hsv, gocv.ColorBGRToHSV)

	thresh := gocv.NewMat()
	defer thresh.Close()

	lb := gocv.NewScalar(0, 200, 200, 1)
	ub := gocv.NewScalar(255, 255, 255, 1)

	gocv.InRangeWithScalar(hsv, lb, ub, &thresh)
	gocv.GaussianBlur(thresh, &thresh, image.Point{11, 11}, 31, 31, gocv.BorderDefault)

	contours, bounds := contoursAndBounds(thresh)
	if len(contours) == 0 {
		return ER, errors.New("No red areas detected to extract mask")
	}

	gocv.FillPoly(&mask, contours, color.RGBA{255, 255, 255, 1})

	roi := ROI{
		Mask:       mask,
		Bounds:     bounds,
		contours:   contours,
		maskRegion: mask.Region(bounds),
	}

	return roi, nil
}

func contoursAndBounds(mat gocv.Mat) ([][]image.Point, image.Rectangle) {
	bounds := image.ZR
	contours := gocv.FindContours(mat, gocv.RetrievalExternal, gocv.ChainApproxSimple)

	for _, c := range contours {
		rect := gocv.BoundingRect(c)
		bounds = bounds.Union(rect)
	}

	return contours, bounds
}

// MaskPreview creates Mat with gray area outside of ROI.
// Result is not resized to ROI bounds - size is equal to input Mat size
func (r *ROI) MaskPreview(src gocv.Mat, dst *gocv.Mat) {
	previewROI := gocv.NewMat()
	defer previewROI.Close()

	gocv.BitwiseAnd(src, r.Mask, &previewROI)
	gocv.AddWeighted(src, 0.3, previewROI, 1, 1, dst)
}

// Apply resizes input Mat to bounds of ROI and applies Mask
func (r *ROI) Apply(src gocv.Mat, dst *gocv.Mat) {
	srcRegion := src.Region(r.Bounds)
	defer srcRegion.Close()

	gocv.BitwiseAnd(r.maskRegion, srcRegion, dst)
}

// Close for close unlaying Mats
func (r *ROI) Close() {
	r.Mask.Close()
	r.maskRegion.Close()
}
