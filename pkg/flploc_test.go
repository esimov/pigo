package pigo_test

import (
	"log"
	"os"
	"runtime"
	"testing"

	pigo "github.com/esimov/pigo/pkg"
	"github.com/stretchr/testify/assert"
)

var flpc []byte

const perturb = 63

func init() {
	flpc, err = os.ReadFile("./cascade/lps/lp42")
	if err != nil {
		log.Fatalf("missing cascade file: %v", err)
	}
}

func TestFlploc_UnpackCascadeFileShouldNotBeNil(t *testing.T) {
	plc, err = pigo.UnpackPlc()

	assert.Nil(t, err)
	assert.NotNil(t, plc)
}

func TestFlploc_LandmarkDetectorShouldReturnDetectionPoints(t *testing.T) {
	p = pigo.UnpackCascade()

	dets := p.RunClassifier(*cParams, 0.0)
	dets = p.ClusterDetections(dets, 0.1)

	landMarkPoints := []pigo.Puploc{}

	for _, det := range dets {
		if det.Scale > 50 {
			// left eye
			puploc := &pigo.Puploc{
				Row:      det.Row - int(0.075*float32(det.Scale)),
				Col:      det.Col - int(0.175*float32(det.Scale)),
				Scale:    float32(det.Scale) * 0.25,
				Perturbs: 50,
			}
			leftEye := plc.RunClassifier(puploc, *imgParams, 0.0, false)

			// right eye
			puploc = &pigo.Puploc{
				Row:      det.Row - int(0.075*float32(det.Scale)),
				Col:      det.Col + int(0.185*float32(det.Scale)),
				Scale:    float32(det.Scale) * 0.25,
				Perturbs: 50,
			}
			rightEye := plc.RunClassifier(puploc, *imgParams, 0.0, false)

			flp := plc.LandmarkPoint(leftEye, rightEye, *imgParams, perturb, false)
			landMarkPoints = append(landMarkPoints, *flp)
		}
	}

	assert.Len(t, landMarkPoints, 1)
}

func TestFlploc_LandmarkDetectorShouldReturnCorrectDetectionPoints(t *testing.T) {
	var (
		eyeCascades   = []string{"lp46", "lp44", "lp42", "lp38", "lp312"}
		mouthCascades = []string{"lp93", "lp84", "lp82", "lp81"}
		flpcs         map[string][]*pigo.FlpCascade

		detected int
	)

	p = pigo.UnpackCascade()

	dets := p.RunClassifier(*cParams, 0.0)
	dets = p.ClusterDetections(dets, 0.1)

	flpcs, err = pigo.ReadFlpCascadeDir()
	assert.ErrorIs(t, err, nil)

	for _, det := range dets {
		if det.Scale > 50 {
			// left eye
			puploc := &pigo.Puploc{
				Row:      det.Row - int(0.075*float32(det.Scale)),
				Col:      det.Col - int(0.175*float32(det.Scale)),
				Scale:    float32(det.Scale) * 0.25,
				Perturbs: 50,
			}
			leftEye := plc.RunClassifier(puploc, *imgParams, 0.0, false)

			// right eye
			puploc = &pigo.Puploc{
				Row:      det.Row - int(0.075*float32(det.Scale)),
				Col:      det.Col + int(0.185*float32(det.Scale)),
				Scale:    float32(det.Scale) * 0.25,
				Perturbs: 50,
			}
			rightEye := plc.RunClassifier(puploc, *imgParams, 0.0, false)

			for _, eye := range eyeCascades {
				for _, flpc := range flpcs[eye] {
					flp := flpc.LandmarkPoint(leftEye, rightEye, *imgParams, perturb, false)
					if flp.Row > 0 && flp.Col > 0 {
						detected++
					}

					flp = flpc.LandmarkPoint(leftEye, rightEye, *imgParams, perturb, true)
					if flp.Row > 0 && flp.Col > 0 {
						detected++
					}
				}
			}
			for _, mouth := range mouthCascades {
				for _, flpc := range flpcs[mouth] {
					flp := flpc.LandmarkPoint(leftEye, rightEye, *imgParams, perturb, false)
					if flp.Row > 0 && flp.Col > 0 {
						detected++
					}
				}
			}

			flp := flpcs["lp84"][0].LandmarkPoint(leftEye, rightEye, *imgParams, perturb, true)
			if flp.Row > 0 && flp.Col > 0 {
				detected++
			}

		}
	}
	// landmark points of the left/right eyes, mouth + nose
	expected := 2*len(eyeCascades) + len(mouthCascades) + 1

	assert.Equal(t, expected, detected)
}

func BenchmarkFlploc_ReadCascadeDir(b *testing.B) {
	for b.Loop() {
		pigo.ReadFlpCascadeDir()
	}
}

func BenchmarkFlploc_LendmarkPoint(b *testing.B) {
	plc, err = pigo.UnpackPlc()
	assert.Nil(b, err)

	cParams.Pixels = pixs

	flploc := &pigo.Puploc{Row: 10, Col: 10, Scale: 20, Perturbs: 50}

	// For benchmarking we are using common values for left and right eye.
	puploc := plc.RunClassifier(flploc, *imgParams, 0.0, false)

	runtime.GC()
	b.ResetTimer()

	for b.Loop() {
		plc.LandmarkPoint(puploc, puploc, *imgParams, 63, false)
	}
}

func BenchmarkFlploc_Detection(b *testing.B) {
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
				leftEye := plc.RunClassifier(puploc, *imgParams, 0.0, false)

				// right eye
				puploc = &pigo.Puploc{
					Row:      det.Row - int(0.075*float32(det.Scale)),
					Col:      det.Col + int(0.185*float32(det.Scale)),
					Scale:    float32(det.Scale) * 0.25,
					Perturbs: 50,
				}
				rightEye := plc.RunClassifier(puploc, *imgParams, 0.0, false)

				plc.LandmarkPoint(leftEye, rightEye, *imgParams, 63, false)

			}
		}
	}
	_ = dets
}
