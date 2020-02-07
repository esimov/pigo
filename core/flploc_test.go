package pigo_test

import (
	"io/ioutil"
	"log"
	"testing"

	pigo "github.com/esimov/pigo/core"
)

var flpc []byte

func init() {
	var err error
	flpc, err = ioutil.ReadFile("../cascade/lps/lp42")
	if err != nil {
		log.Fatalf("missing cascade file: %v", err)
	}
}

func TestFlploc_UnpackCascadeFileShouldNotBeNil(t *testing.T) {
	var (
		err error
		pl  = pigo.NewPuplocCascade()
	)
	plc, err = pl.UnpackCascade(flpc)
	if err != nil {
		t.Fatalf("failed unpacking the cascade file: %v", err)
	}
}

func TestFlploc_LandmarkPointsFinderShouldReturnDetectionPoints(t *testing.T) {
	p := pigo.NewPigo()
	// Unpack the binary file. This will return the number of cascade trees,
	// the tree depth, the threshold and the prediction from tree's leaf nodes.
	classifier, err := p.Unpack(pigoCascade)
	if err != nil {
		t.Fatalf("error reading the cascade file: %s", err)
	}

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	faces := classifier.RunCascade(*cParams, 0.0)
	// Calculate the intersection over union (IoU) of two clusters.
	faces = classifier.ClusterDetections(faces, 0.1)

	landMarkPoints := []pigo.Puploc{}

	for _, face := range faces {
		if face.Scale > 50 {
			// left eye
			puploc := &pigo.Puploc{
				Row:      face.Row - int(0.075*float32(face.Scale)),
				Col:      face.Col - int(0.175*float32(face.Scale)),
				Scale:    float32(face.Scale) * 0.25,
				Perturbs: 50,
			}
			leftEye := plc.RunDetector(*puploc, *imgParams, 0.0, false)

			// right eye
			puploc = &pigo.Puploc{
				Row:      face.Row - int(0.075*float32(face.Scale)),
				Col:      face.Col + int(0.185*float32(face.Scale)),
				Scale:    float32(face.Scale) * 0.25,
				Perturbs: 50,
			}
			rightEye := plc.RunDetector(*puploc, *imgParams, 0.0, false)

			flp := plc.FindLandmarkPoints(leftEye, rightEye, *imgParams, 63, false)
			landMarkPoints = append(landMarkPoints, *flp)
		}
	}
	if len(landMarkPoints) == 0 {
		t.Fatalf("should have been detected facial landmark points: %s", err)
	}
}

func BenchmarkFlploc(b *testing.B) {
	pg := pigo.NewPigo()
	// Unpack the binary file. This will return the number of cascade trees,
	// the tree depth, the threshold and the prediction from tree's leaf nodes.
	classifier, err := pg.Unpack(pigoCascade)
	if err != nil {
		b.Fatalf("error reading the cascade file: %s", err)
	}

	pl := pigo.PuplocCascade{}
	plc, err := pl.UnpackCascade(puplocCascade)
	if err != nil {
		b.Fatalf("error reading the cascade file: %s", err)
	}

	var faces []pigo.Detection

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		pixs := pigo.RgbToGrayscale(srcImg)
		cParams.Pixels = pixs
		// Run the classifier over the obtained leaf nodes and return the detection results.
		// The result contains quadruplets representing the row, column, scale and detection score.
		faces = classifier.RunCascade(*cParams, 0.0)
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
				leftEye := plc.RunDetector(*puploc, *imgParams, 0.0, false)

				// right eye
				puploc = &pigo.Puploc{
					Row:      face.Row - int(0.075*float32(face.Scale)),
					Col:      face.Col + int(0.185*float32(face.Scale)),
					Scale:    float32(face.Scale) * 0.25,
					Perturbs: 50,
				}
				rightEye := plc.RunDetector(*puploc, *imgParams, 0.0, false)

				plc.FindLandmarkPoints(leftEye, rightEye, *imgParams, 63, false)

			}
		}
	}
	_ = faces
}
