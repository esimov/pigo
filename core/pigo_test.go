package pigo_test

import (
	"image"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	pigo "github.com/esimov/pigo/core"
)

var (
	p        = pigo.NewPigo()
	err      error
	faceCasc []byte
	pixs     []uint8
	srcImg   *image.NRGBA
)

func init() {
	faceCasc, err = os.ReadFile("../cascade/facefinder")
	if err != nil {
		log.Fatalf("Error reading the cascade file: %v", err)
	}

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
	p, err = p.Unpack(faceCasc)
	if err != nil {
		t.Fatalf("failed unpacking the cascade file: %v", err)
	}
}

func TestPigo_InputImageShouldBeGrayscale(t *testing.T) {
	// Since an image converted grayscale has only one channel,we should assume
	// that the grayscale image array length is the source image length / 4.
	if len(imgParams.Pixels) != len(srcImg.Pix)/4 {
		t.Fatalf("the source image should be converted to grayscale")
	}
}

func TestPigo_Detector_ShouldDetectFace(t *testing.T) {
	// Unpack the facefinder binary cascade file. This will return the number of cascade trees,
	// the tree depth, the threshold and the prediction from tree's leaf nodes.
	p, err = p.Unpack(faceCasc)
	if err != nil {
		t.Fatalf("error reading the cascade file: %s", err)
	}

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	dets := p.RunCascade(*cParams, 0.0)
	// Calculate the intersection over union (IoU) of two clusters.
	dets = p.ClusterDetections(dets, 0.1)
	if len(dets) == 0 {
		t.Fatalf("face should've been detected")
	}
}

func BenchmarkPigoUnpackCascade(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// Unpack the facefinder binary cascade file.
		_, err := p.Unpack(faceCasc)
		if err != nil {
			log.Fatalf("error reading the cascade file: %s", err)
		}
	}
}

func BenchmarkPigoFaceDetection(b *testing.B) {
	var dets []pigo.Detection

	p, err = p.Unpack(faceCasc)
	if err != nil {
		log.Fatalf("error reading the cascade file: %s", err)
	}

	pixs := pigo.RgbToGrayscale(srcImg)
	cParams.Pixels = pixs

	runtime.GC()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Run the classifier over the obtained leaf nodes and return the detection results.
		// The result contains quadruplets representing the row, column, scale and detection score.
		dets = p.RunCascade(*cParams, 0.0)
		// Calculate the intersection over union (IoU) of two clusters.
		dets = p.ClusterDetections(dets, 0.1)
	}
	_ = dets
}

func BenchmarkPigoClusterDetection(b *testing.B) {
	var dets []pigo.Detection

	p, err = p.Unpack(faceCasc)
	if err != nil {
		log.Fatalf("error reading the cascade file: %s", err)
	}

	pixs := pigo.RgbToGrayscale(srcImg)
	cParams.Pixels = pixs

	runtime.GC()
	b.ResetTimer()

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	dets = p.RunCascade(*cParams, 0.0)

	for i := 0; i < b.N; i++ {
		// Calculate the intersection over union (IoU) of two clusters.
		dets = p.ClusterDetections(dets, 0.1)
	}
	_ = dets
}
