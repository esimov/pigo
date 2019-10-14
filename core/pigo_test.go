package pigo_test

import (
	"image"
	"io/ioutil"
	"log"
	"path/filepath"
	"testing"

	pigo "github.com/esimov/pigo/core"
)

var (
	pigoCascade []byte
	srcImg      *image.NRGBA
)

func init() {
	var err error
	pigoCascade, err = ioutil.ReadFile("../cascade/facefinder")
	if err != nil {
		log.Fatalf("Error reading the cascade file: %v", err)
	}

	source := filepath.Join("../testdata", "sample.jpg")
	srcImg, err = pigo.GetImage(source)
	if err != nil {
		log.Fatalf("error reading the source file: %s", err)
	}

	pixs := pigo.RgbToGrayscale(srcImg)
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

func BenchmarkPigo(b *testing.B) {
	pg := pigo.NewPigo()
	// Unpack the binary file. This will return the number of cascade trees,
	// the tree depth, the threshold and the prediction from tree's leaf nodes.
	classifier, err := pg.Unpack(pigoCascade)
	if err != nil {
		log.Fatalf("Error reading the cascade file: %s", err)
	}

	var dets []pigo.Detection
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		pixs := pigo.RgbToGrayscale(srcImg)
		cParams.Pixels = pixs
		// Run the classifier over the obtained leaf nodes and return the detection results.
		// The result contains quadruplets representing the row, column, scale and detection score.
		dets = classifier.RunCascade(*cParams, 0.0)
		// Calculate the intersection over union (IoU) of two clusters.
		dets = classifier.ClusterDetections(dets, 0.1)
	}
	_ = dets
}
