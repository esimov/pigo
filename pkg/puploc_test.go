package pigo_test

import (
	"log"
	"os"
	"runtime"
	"testing"

	pigo "github.com/esimov/pigo/pkg"
	"github.com/stretchr/testify/assert"
)

var (
	plc        *pigo.PuplocCascade
	puplocCasc []byte
	imgParams  *pigo.ImageParams
	cParams    *pigo.CascadeParams
)

func init() {
	puplocCasc, err = os.ReadFile("./cascade/puploc")
	if err != nil {
		log.Fatalf("error reading the puploc cascade file: %v", err)
	}
}

func TestPuploc_UnpackCascadeFileShouldNotBeNil(t *testing.T) {
	plc, err = pigo.UnpackPlc()
	assert.Nil(t, err)
	assert.NotNil(t, plc)
}

func TestPuploc_Detector_ShouldDetectEyes(t *testing.T) {
	p = pigo.UnpackCascade()
	plc, err = pigo.UnpackPlc()
	assert.Nil(t, err)

	dets := p.RunClassifier(*cParams, 0.0)
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
			plc.RunClassifier(puploc, imgParams, 0.0, false)

			// right eye
			puploc = &pigo.Puploc{
				Row:      det.Row - int(0.075*float32(det.Scale)),
				Col:      det.Col + int(0.185*float32(det.Scale)),
				Scale:    float32(det.Scale) * 0.25,
				Perturbs: 50,
			}
			plc.RunClassifier(puploc, imgParams, 0.0, false)
			eyes = append(eyes, *puploc)
		}
	}

	assert.NotEqual(t, 0, len(eyes))
}

func BenchmarkPuploc_UnpackCascade(b *testing.B) {
	p = pigo.UnpackCascade()

	runtime.GC()
	b.ResetTimer()

	for b.Loop() {
		// Read and unpack the pupil localization cascade file.
		plc, _ = pigo.UnpackPlc()
	}
}

func BenchmarkPuploc_DetectorRun(b *testing.B) {
	plc, err = pigo.UnpackPlc()
	assert.Nil(b, err)

	cParams.Pixels = pixs
	puploc := &pigo.Puploc{Row: 10, Col: 10, Scale: 20, Perturbs: 50}

	for b.Loop() {
		plc.RunClassifier(puploc, imgParams, 0.0, false)
	}
}

func BenchmarkPuploc_Detection(b *testing.B) {
	p = pigo.UnpackCascade()
	plc, err = pigo.UnpackPlc()
	assert.Nil(b, err)

	cParams.Pixels = pixs

	runtime.GC()
	b.ResetTimer()

	dets := p.RunClassifier(*cParams, 0.0)
	dets = p.ClusterDetections(dets, 0.1)

	for b.Loop() {
		for _, det := range dets {
			if det.Scale > 50 {
				// left eye
				puploc := &pigo.Puploc{
					Row:      det.Row - int(0.075*float32(det.Scale)),
					Col:      det.Col - int(0.175*float32(det.Scale)),
					Scale:    float32(det.Scale) * 0.25,
					Perturbs: 50,
				}
				plc.RunClassifier(puploc, imgParams, 0.0, false)

				// right eye
				puploc = &pigo.Puploc{
					Row:      det.Row - int(0.075*float32(det.Scale)),
					Col:      det.Col + int(0.185*float32(det.Scale)),
					Scale:    float32(det.Scale) * 0.25,
					Perturbs: 50,
				}
				plc.RunClassifier(puploc, imgParams, 0.0, false)
			}
		}
	}
	_ = dets
}
