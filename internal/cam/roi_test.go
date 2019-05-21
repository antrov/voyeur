package cam

import (
	"image/color"
	"io/ioutil"
	"testing"

	"gocv.io/x/gocv"
)

func TestNewROI(t *testing.T) {
	roi, err := NewROI("/Users/antrov/go/src/gitlab.com/antrov/couch-watch/resources/testing/test_mask.png")
	defer roi.Close()

	if err != nil {
		t.Errorf("%s", err)
	}

	if roi.Bounds.Empty() {
		t.Error("Mask bounds are empty")
	}

	if len(roi.contours) != 1 {
		t.Errorf("Mask areas count is invalid (1 expected, was %d)", len(roi.contours))
	}

	gocv.Rectangle(&roi.Mask, roi.Bounds, color.RGBA{255, 0, 0, 1}, 3)

	ok := gocv.IMWrite("/Users/antrov/go/src/gitlab.com/antrov/couch-watch/out/roi_from_file.png", roi.Mask)
	if !ok {
		t.Error("Result image cannot be saved")
	}
}

func TestNewROIFromDrawing(t *testing.T) {
	bytes, err := ioutil.ReadFile("/Users/antrov/go/src/gitlab.com/antrov/couch-watch/resources/testing/roi_test_draw.jpg")
	if err != nil {
		t.Errorf("%s", err)
	}

	roi, err := NewROIFromDrawing(bytes)
	defer roi.Close()

	if err != nil {
		t.Errorf("%s", err)
	}

	if roi.Bounds.Empty() {
		t.Error("Mask bounds are empty")
	}

	if len(roi.contours) != 3 {
		t.Errorf("Mask areas count is invalid (3 expected, was %d)", len(roi.contours))
	}

	gocv.Rectangle(&roi.Mask, roi.Bounds, color.RGBA{255, 0, 0, 1}, 1)

	ok := gocv.IMWrite("/Users/antrov/go/src/gitlab.com/antrov/couch-watch/out/roi_from_drawing.png", roi.Mask)
	if !ok {
		t.Error("Result image cannot be saved")
	}
}

func TestApply(t *testing.T) {
	roi, err := NewROI("/Users/antrov/go/src/gitlab.com/antrov/couch-watch/resources/testing/test_mask.png")
	defer roi.Close()

	if err != nil {
		t.Errorf("%s", err)
	}

	if roi.Bounds.Empty() {
		t.Error("Mask bounds are empty")
	}

	img := gocv.IMRead("/Users/antrov/go/src/gitlab.com/antrov/couch-watch/resources/testing/roi_test_draw.jpg", gocv.IMReadColor)
	if img.Empty() {
		t.Error("Loaded image for maskign is empty")
	}

	masked := gocv.NewMat()
	defer masked.Close()

	roi.Apply(img, &masked)

	ok := gocv.IMWrite("/Users/antrov/go/src/gitlab.com/antrov/couch-watch/out/roi_masked.png", masked)
	if !ok {
		t.Error("Result image cannot be saved")
	}
}

func TestMaskPreview(t *testing.T) {
	roi, err := NewROI("/Users/antrov/go/src/gitlab.com/antrov/couch-watch/resources/testing/test_mask.png")
	defer roi.Close()

	if err != nil {
		t.Errorf("%s", err)
	}

	if roi.Bounds.Empty() {
		t.Error("Mask bounds are empty")
	}

	img := gocv.IMRead("/Users/antrov/go/src/gitlab.com/antrov/couch-watch/resources/testing/roi_test_draw.jpg", gocv.IMReadColor)
	if img.Empty() {
		t.Error("Loaded image for maskign is empty")
	}

	preview := gocv.NewMat()
	defer preview.Close()

	roi.MaskPreview(img, &preview)

	ok := gocv.IMWrite("/Users/antrov/go/src/gitlab.com/antrov/couch-watch/out/roi_preview.png", preview)
	if !ok {
		t.Error("Result image cannot be saved")
	}
}
