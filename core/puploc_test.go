package pigo_test

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"testing"

	pigo "github.com/esimov/pigo/core"
)

var (
	pigoCascade   []byte
	puplocCascade []byte
)

func init() {
	var err error
	pigoCascade, err = ioutil.ReadFile("../cascade/facefinder")
	if err != nil {
		log.Fatalf("Error reading the cascade file: %v", err)
	}

	puplocCascade, err = ioutil.ReadFile("../cascade/puploc")
	if err != nil {
		log.Fatalf("Error reading the puploc cascade file: %v", err)
	}
}

func BenchmarkPuploc(b *testing.B) {
	source := filepath.Join("../testdata", "sample.jpg")
	src, err := pigo.GetImage(source)
	if err != nil {
		log.Fatalf("Error reading the source file: %s", err)
	}

	pixs := pigo.RgbToGrayscale(src)
	cols, rows := src.Bounds().Max.X, src.Bounds().Max.Y

	imgParams := pigo.ImageParams{
		Pixels: pixs,
		Rows:   rows,
		Cols:   cols,
		Dim:    cols,
	}

	cParams := pigo.CascadeParams{
		MinSize:     20,
		MaxSize:     1000,
		ShiftFactor: 0.2,
		ScaleFactor: 1.1,
		ImageParams: imgParams,
	}
	pg := pigo.NewPigo()
	// Unpack the binary file. This will return the number of cascade trees,
	// the tree depth, the threshold and the prediction from tree's leaf nodes.
	classifier, err := pg.Unpack(pigoCascade)
	if err != nil {
		log.Fatalf("Error reading the cascade file: %s", err)
	}

	pl := pigo.PuplocCascade{}
	plc, err := pl.UnpackCascade(puplocCascade)
	if err != nil {
		log.Fatalf("Error reading the cascade file: %s", err)
	}

	var faces []pigo.Detection

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		pixs := pigo.RgbToGrayscale(src)
		cParams.Pixels = pixs
		// Run the classifier over the obtained leaf nodes and return the detection results.
		// The result contains quadruplets representing the row, column, scale and detection score.
		faces = classifier.RunCascade(cParams, 0.0)
		// Calculate the intersection over union (IoU) of two clusters.
		faces = classifier.ClusterDetections(faces, 0.1)

		for _, face := range faces {
			if face.Scale > 50 {
				// left eye
				puploc := &pigo.Puploc{
					Row:      face.Row - int(0.075*float32(face.Scale)),
					Col:      face.Col - int(0.175*float32(face.Scale)),
					Scale:    float32(face.Scale) * 0.25,
					Perturbs: 50,
				}
				plc.RunDetector(*puploc, imgParams, 0.0)

				// right eye
				puploc = &pigo.Puploc{
					Row:      face.Row - int(0.075*float32(face.Scale)),
					Col:      face.Col + int(0.185*float32(face.Scale)),
					Scale:    float32(face.Scale) * 0.25,
					Perturbs: 50,
				}
				plc.RunDetector(*puploc, imgParams, 0.0)
			}
		}
	}
	_ = faces
}
