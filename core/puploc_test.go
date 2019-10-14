package pigo_test

import (
	"io/ioutil"
	"log"
	"testing"

	pigo "github.com/esimov/pigo/core"
)

var (
	puplocCascade []byte
	plc           *pigo.PuplocCascade
	imgParams     *pigo.ImageParams
	cParams       *pigo.CascadeParams
)

func init() {
	var err error
	puplocCascade, err = ioutil.ReadFile("../cascade/puploc")
	if err != nil {
		log.Fatalf("error reading the puploc cascade file: %v", err)
	}
}

func TestPuploc_UnpackCascadeFileShouldNotBeNil(t *testing.T) {
	var (
		err error
		pl  = pigo.NewPuplocCascade()
	)
	plc, err = pl.UnpackCascade(puplocCascade)
	if err != nil {
		t.Fatalf("failed unpacking the cascade file: %v", err)
	}
}

func TestPuploc_Detector_ShouldDetectEyes(t *testing.T) {
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

	eyes := []pigo.Puploc{}

	for _, face := range faces {
		if face.Scale > 50 {
			// left eye
			puploc := &pigo.Puploc{
				Row:      face.Row - int(0.075*float32(face.Scale)),
				Col:      face.Col - int(0.175*float32(face.Scale)),
				Scale:    float32(face.Scale) * 0.25,
				Perturbs: 50,
			}
			plc.RunDetector(*puploc, *imgParams, 0.0, false)

			// right eye
			puploc = &pigo.Puploc{
				Row:      face.Row - int(0.075*float32(face.Scale)),
				Col:      face.Col + int(0.185*float32(face.Scale)),
				Scale:    float32(face.Scale) * 0.25,
				Perturbs: 50,
			}
			plc.RunDetector(*puploc, *imgParams, 0.0, false)
			eyes = append(eyes, *puploc)
		}
	}

	if len(eyes) == 0 {
		t.Fatalf("should have been detected eyes: %s", err)
	}
}

func BenchmarkPuploc(b *testing.B) {
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
				plc.RunDetector(*puploc, *imgParams, 0.0, false)

				// right eye
				puploc = &pigo.Puploc{
					Row:      face.Row - int(0.075*float32(face.Scale)),
					Col:      face.Col + int(0.185*float32(face.Scale)),
					Scale:    float32(face.Scale) * 0.25,
					Perturbs: 50,
				}
				plc.RunDetector(*puploc, *imgParams, 0.0, false)
			}
		}
	}
	_ = faces
}
