package cam

import (
	"errors"
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

func createMask(base, draw gocv.Mat, mask, preview *gocv.Mat) (err error) {
	if mask == nil {
		return errors.New("mask var have to be initalized before calling func")
	}

	back := gocv.NewMatWithSize(draw.Size()[0], draw.Size()[1], gocv.MatTypeCV8UC3)
	back.CopyTo(mask)
	back.Close()

	hsv := gocv.NewMat()
	defer hsv.Close()

	gocv.CvtColor(draw, &hsv, gocv.ColorBGRToHSV)

	thresh := gocv.NewMat()
	defer thresh.Close()

	lb := gocv.NewScalar(0, 200, 200, 1)
	ub := gocv.NewScalar(255, 255, 255, 1)

	gocv.InRangeWithScalar(hsv, lb, ub, &thresh)

	gocv.GaussianBlur(thresh, &thresh, image.Point{11, 11}, 31, 31, gocv.BorderDefault)

	// maskPrev := gocv.NewMatWithSize(base.Size()[0], base.Size()[1], base.Type())
	// defer maskPrev.Close()

	contours := gocv.FindContours(thresh, gocv.RetrievalExternal, gocv.ChainApproxSimple)
	gocv.FillPoly(mask, contours, color.RGBA{255, 255, 255, 1})
	// gocv.FillPoly(&maskPrev, contours, color.RGBA{255, 255, 255, 1})

	roi := gocv.NewMat()
	defer roi.Close()

	gocv.BitwiseAnd(base, *mask, &roi)
	gocv.AddWeighted(base, 0.3, roi, 1, 0, preview)

	return nil
}

// func main() {
// 	// window := gocv.NewWindow("Motion Window")
// 	// window.SetWindowProperty(gocv.WindowPropertyFullscreen, gocv.WindowFullscreen)
// 	// window.SetWindowProperty(gocv.WindowPropertyFullscreen, gocv.WindowNormal)
// 	// defer window.Close()

// 	src := gocv.IMRead("/Users/antrov/Projekty/anti-couchsurfer/roi_src.jpg", gocv.IMReadColor)
// 	defer src.Close()

// 	img := gocv.IMRead("/Users/antrov/Projekty/anti-couchsurfer/roi.jpg", gocv.IMReadColor)
// 	defer img.Close()

// 	mask := gocv.NewMat()
// 	defer mask.Close()

// 	preview := gocv.NewMat()
// 	defer preview.Close()

// 	createMask(img, img, &mask, &preview)

// 	gocv.IMWrite("out.png", preview)
// 	// window.IMShow(mask)
// 	// window.IMShow(preview)

// 	// if window.WaitKey(10000) == 27 {
// 	// 	return
// 	// }
// }
