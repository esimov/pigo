package pigo_test

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"testing"

	pigo "github.com/esimov/pigo/core"
)

var pigoCascadeFile []byte

func init() {
	var err error
	pigoCascadeFile, err = ioutil.ReadFile("../cascade/facefinder")
	if err != nil {
		log.Fatalf("Error reading the cascade file: %v", err)
	}
}

func BenchmarkPigo(b *testing.B) {
	source := filepath.Join("../testdata", "sample.jpg")
	src, err := pigo.GetImage(source)
	if err != nil {
		log.Fatalf("Error reading the source file: %s", err)
	}

	pixs := pigo.RgbToGrayscale(src)
	cols, rows := src.Bounds().Max.X, src.Bounds().Max.Y

	cParams := pigo.CascadeParams{
		MinSize:     20,
		MaxSize:     1000,
		ShiftFactor: 0.2,
		ScaleFactor: 1.1,
		ImageParams: pigo.ImageParams{
			Pixels: pixs,
			Rows:   rows,
			Cols:   cols,
			Dim:    cols,
		},
	}
	pg := pigo.NewPigo()
	// Unpack the binary file. This will return the number of cascade trees,
	// the tree depth, the threshold and the prediction from tree's leaf nodes.
	classifier, err := pg.Unpack(pigoCascadeFile)
	if err != nil {
		log.Fatalf("Error reading the cascade file: %s", err)
	}

	var dets []pigo.Detection
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		pixs := pigo.RgbToGrayscale(src)
		cParams.Pixels = pixs
		// Run the classifier over the obtained leaf nodes and return the detection results.
		// The result contains quadruplets representing the row, column, scale and detection score.
		dets = classifier.RunCascade(cParams, 0.0)
		// Calculate the intersection over union (IoU) of two clusters.
		dets = classifier.ClusterDetections(dets, 0.1)
	}
	_ = dets
}
