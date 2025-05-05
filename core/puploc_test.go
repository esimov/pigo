package pigo_test

import (
	"log"
	"os"
	"runtime"
	"testing"

	pigo "github.com/esimov/pigo/core"
)

var (
	pl         = pigo.NewPuplocCascade()
	puplocCasc []byte
	plc        *pigo.PuplocCascade
	imgParams  *pigo.ImageParams
	cParams    *pigo.CascadeParams
)

func init() {
	puplocCasc, err = os.ReadFile("../cascade/puploc")
	if err != nil {
		log.Fatalf("error reading the puploc cascade file: %v", err)
	}
}

func TestPuploc_UnpackCascadeFileShouldNotBeNil(t *testing.T) {
	plc, err = pl.UnpackCascade(puplocCasc)
	if err != nil {
		t.Fatalf("failed unpacking the cascade file: %v", err)
	}
}

func TestPuploc_Detector_ShouldDetectEyes(t *testing.T) {
	// Unpack the facefinder binary cascade file. This will return the number of cascade trees,
	// the tree depth, the threshold and the prediction from tree's leaf nodes.
	p, err = p.Unpack(faceCasc)
	if err != nil {
		t.Fatalf("error reading the cascade file: %s", err)
	}

	plc, err = pl.UnpackCascade(puplocCasc)
	if err != nil {
		t.Fatalf("failed unpacking the cascade file: %v", err)
	}

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	dets := p.RunCascade(*cParams, 0.0)
	// Calculate the intersection over union (IoU) of two clusters.
	dets = p.ClusterDetections(dets, 0.1)

	eyes := []pigo.Puploc{}

	for _, det := range dets {
		if det.Scale > 50 {
			// left eye
			puploc := &pigo.Puploc{
				Row:      det.Row - int(0.075*float32(det.Scale)),
				Col:      det.Col - int(0.175*float32(det.Scale)),
				Scale:    float32(det.Scale) * 0.25,
				Perturbs: 50,
			}
			plc.RunDetector(*puploc, *imgParams, 0.0, false)

			// right eye
			puploc = &pigo.Puploc{
				Row:      det.Row - int(0.075*float32(det.Scale)),
				Col:      det.Col + int(0.185*float32(det.Scale)),
				Scale:    float32(det.Scale) * 0.25,
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
	// Unpack the facefinder binary cascade file.
	_, err := p.Unpack(faceCasc)
	if err != nil {
		log.Fatalf("error reading the cascade file: %s", err)
	}

	runtime.GC()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Unpack the pupil localization cascade file.
		_, err = pl.UnpackCascade(puplocCasc)
		if err != nil {
			b.Fatalf("error reading the cascade file: %s", err)
		}
	}
}

func BenchmarkPuplocDetectorRun(b *testing.B) {
	plc, err := pl.UnpackCascade(puplocCasc)
	if err != nil {
		b.Fatalf("error reading the cascade file: %s", err)
	}

	cParams.Pixels = pixs

	puploc := &pigo.Puploc{Row: 10, Col: 10, Scale: 20, Perturbs: 50}
	for i := 0; i < b.N; i++ {
		plc.RunDetector(*puploc, *imgParams, 0.0, false)
	}
}

func BenchmarkPuplocDetection(b *testing.B) {
	p, err = p.Unpack(faceCasc)
	if err != nil {
		b.Fatalf("error reading the cascade file: %s", err)
	}

	plc, err = pl.UnpackCascade(puplocCasc)
	if err != nil {
		b.Fatalf("error reading the cascade file: %s", err)
	}

	cParams.Pixels = pixs

	runtime.GC()
	b.ResetTimer()

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	dets := p.RunCascade(*cParams, 0.0)
	// Calculate the intersection over union (IoU) of two clusters.
	dets = p.ClusterDetections(dets, 0.1)

	for i := 0; i < b.N; i++ {
		for _, det := range dets {
			if det.Scale > 50 {
				// left eye
				puploc := &pigo.Puploc{
					Row:      det.Row - int(0.075*float32(det.Scale)),
					Col:      det.Col - int(0.175*float32(det.Scale)),
					Scale:    float32(det.Scale) * 0.25,
					Perturbs: 50,
				}
				plc.RunDetector(*puploc, *imgParams, 0.0, false)

				// right eye
				puploc = &pigo.Puploc{
					Row:      det.Row - int(0.075*float32(det.Scale)),
					Col:      det.Col + int(0.185*float32(det.Scale)),
					Scale:    float32(det.Scale) * 0.25,
					Perturbs: 50,
				}
				plc.RunDetector(*puploc, *imgParams, 0.0, false)
			}
		}
	}
	_ = dets
}
