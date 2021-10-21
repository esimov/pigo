package pigo_test

import (
	"io/ioutil"
	"log"
	"runtime"
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
	// Unpack the facefinder binary cascade file. This will return the number of cascade trees,
	// the tree depth, the threshold and the prediction from tree's leaf nodes.
	p, err := p.Unpack(pigoCascade)
	if err != nil {
		t.Fatalf("error reading the cascade file: %s", err)
	}

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	faces := p.RunCascade(*cParams, 0.0)
	// Calculate the intersection over union (IoU) of two clusters.
	faces = p.ClusterDetections(faces, 0.1)

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
		t.Fatalf("eyes should've been detected")
	}
}

func BenchmarkPuplocUnpackCascade(b *testing.B) {
	pg := pigo.NewPigo()

	// Unpack the facefinder binary cascade file.
	_, err := pg.Unpack(pigoCascade)
	if err != nil {
		log.Fatalf("error reading the cascade file: %s", err)
	}

	b.ResetTimer()
	runtime.GC()

	for i := 0; i < b.N; i++ {
		pl := pigo.PuplocCascade{}
		// Unpack the pupil localization cascade file.
		_, err = pl.UnpackCascade(puplocCascade)
		if err != nil {
			b.Fatalf("error reading the cascade file: %s", err)
		}
	}
}

func BenchmarkPuplocDetectorRun(b *testing.B) {
	pl := pigo.PuplocCascade{}

	plc, err := pl.UnpackCascade(puplocCascade)
	if err != nil {
		b.Fatalf("error reading the cascade file: %s", err)
	}

	pixs := pigo.RgbToGrayscale(srcImg)
	cParams.Pixels = pixs

	puploc := &pigo.Puploc{Row: 10, Col: 10, Scale: 20, Perturbs: 50}
	for i := 0; i < b.N; i++ {
		plc.RunDetector(*puploc, *imgParams, 0.0, false)
	}
}

func BenchmarkPuplocDetection(b *testing.B) {
	var faces []pigo.Detection

	pg := pigo.NewPigo()
	p, err := pg.Unpack(pigoCascade)
	if err != nil {
		b.Fatalf("error reading the cascade file: %s", err)
	}

	pl := pigo.PuplocCascade{}
	plc, err := pl.UnpackCascade(puplocCascade)
	if err != nil {
		b.Fatalf("error reading the cascade file: %s", err)
	}

	pixs := pigo.RgbToGrayscale(srcImg)
	cParams.Pixels = pixs

	b.ResetTimer()
	runtime.GC()

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	faces = p.RunCascade(*cParams, 0.0)
	// Calculate the intersection over union (IoU) of two clusters.
	faces = p.ClusterDetections(faces, 0.1)

	for i := 0; i < b.N; i++ {
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
