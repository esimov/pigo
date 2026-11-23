package pigo_test

import (
	"image"
	"log"
	"path/filepath"
	"runtime"
	"testing"

	pigo "github.com/esimov/pigo/pkg"
	"github.com/stretchr/testify/assert"
)

var (
	classifier *pigo.Pigo
	err        error
	pixs       []uint8
	srcImg     *image.NRGBA
)

func init() {
	source := filepath.Join("../testdata", "sample.jpg")
	srcImg, err = pigo.GetImage(source)
	if err != nil {
		log.Fatalf("error reading the source file: %s", err)
	}

	pixs = pigo.RgbToGrayscale(srcImg)
	cols, rows := srcImg.Bounds().Max.X, srcImg.Bounds().Max.Y

	imgParams = &pigo.ImageParams{
		Pixels: pixs,
		Rows:   rows,
		Cols:   cols,
		Dim:    cols,
	}

	cParams = &pigo.CascadeParams{
		MinSize:     20,
		MaxSize:     1000,
		ShiftFactor: 0.2,
		ScaleFactor: 1.1,
		ImageParams: *imgParams,
	}
}

func TestPigo_UnpackCascadeFileShouldNotBeNil(t *testing.T) {
	classifier = pigo.UnpackCascade()

	assert.NotNil(t, classifier)
}

func TestPigo_InputImageShouldBeGrayscale(t *testing.T) {
	// Since a grayscale image has one channel, its pixel array should be
	// one quarter of the source RGBA image pixel array.
	assert.Len(t, imgParams.Pixels, len(srcImg.Pix)/4)
}

func TestPigo_Detector_ShouldDetectFace(t *testing.T) {
	classifier = pigo.UnpackCascade()

	dets := classifier.RunClassifier(cParams, 0.0)
	dets = classifier.ClusterDetections(dets, 0.1)

	assert.NotEmpty(t, dets)
}

func BenchmarkPigoUnpackCascade(b *testing.B) {
	for b.Loop() {
		// Unpack the facefinder binary cascade file.
		_ = pigo.UnpackCascade()
	}
}

func BenchmarkPigoFaceDetection(b *testing.B) {
	var dets []pigo.Detection

	classifier = pigo.UnpackCascade()

	pixs := pigo.RgbToGrayscale(srcImg)
	cParams.Pixels = pixs

	runtime.GC()
	b.ResetTimer()

	for b.Loop() {
		// Run the classifier over the obtained leaf nodes and return the detection results.
		// The result contains quadruplets representing the row, column, scale and detection score.
		dets = classifier.RunClassifier(cParams, 0.0)
		// Calculate the intersection over union (IoU) of two clusters.
		dets = classifier.ClusterDetections(dets, 0.1)
	}
	_ = dets
}

func BenchmarkPigoClusterDetection(b *testing.B) {
	var dets []pigo.Detection

	classifier = pigo.UnpackCascade()

	pixs := pigo.RgbToGrayscale(srcImg)
	cParams.Pixels = pixs

	runtime.GC()
	b.ResetTimer()

	dets = classifier.RunClassifier(cParams, 0.0)

	for b.Loop() {
		dets = classifier.ClusterDetections(dets, 0.1)
	}
	_ = dets
}
