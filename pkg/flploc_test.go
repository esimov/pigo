package pigo_test

import (
	"runtime"
	"testing"

	pigo "github.com/esimov/pigo/pkg"
	"github.com/stretchr/testify/assert"
)

const perturb = 63

func TestFlploc_UnpackCascadeFileShouldNotBeNil(t *testing.T) {
	plc, err = pigo.UnpackPlc()

	assert.Nil(t, err)
	assert.NotNil(t, plc)
}

func TestFlploc_LandmarkDetectorShouldReturnDetectionPoints(t *testing.T) {
	classifier = pigo.UnpackCascade()

	dets := classifier.RunClassifier(cParams, 0.0)
	dets = classifier.ClusterDetections(dets, 0.1)

	landMarkPoints := []pigo.Puploc{}

	for _, det := range dets {
		// left eye
		puploc := &pigo.Puploc{
			Row:      det.Row - int(0.075*float32(det.Scale)),
			Col:      det.Col - int(0.175*float32(det.Scale)),
			Scale:    float32(det.Scale) * 0.25,
			Perturbs: perturb,
		}
		leftEye := plc.RunClassifier(puploc, imgParams, 0.0, false)

		// right eye
		puploc = &pigo.Puploc{
			Row:      det.Row - int(0.075*float32(det.Scale)),
			Col:      det.Col + int(0.185*float32(det.Scale)),
			Scale:    float32(det.Scale) * 0.25,
			Perturbs: perturb,
		}
		rightEye := plc.RunClassifier(puploc, imgParams, 0.0, false)

		flp := plc.LandmarkPoint(leftEye, rightEye, imgParams, perturb, false)
		landMarkPoints = append(landMarkPoints, *flp)
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

	classifier = pigo.UnpackCascade()

	dets := classifier.RunClassifier(cParams, 0.0)
	dets = classifier.ClusterDetections(dets, 0.1)
	assert.Len(t, dets, 1)

	flpcs, err = pigo.ReadFlpCascadeDir()
	assert.ErrorIs(t, err, nil)

	for _, det := range dets {
		// left eye
		puploc := &pigo.Puploc{
			Row:      det.Row - int(0.075*float32(det.Scale)),
			Col:      det.Col - int(0.175*float32(det.Scale)),
			Scale:    float32(det.Scale) * 0.25,
			Perturbs: perturb,
		}
		leftEye := plc.RunClassifier(puploc, imgParams, 0.0, false)

		// right eye
		puploc = &pigo.Puploc{
			Row:      det.Row - int(0.075*float32(det.Scale)),
			Col:      det.Col + int(0.185*float32(det.Scale)),
			Scale:    float32(det.Scale) * 0.25,
			Perturbs: perturb,
		}
		rightEye := plc.RunClassifier(puploc, imgParams, 0.0, false)

		for _, eye := range eyeCascades {
			for _, flpc := range flpcs[eye] {
				flp := flpc.LandmarkPoint(leftEye, rightEye, imgParams, perturb, false)
				if flp != nil {
					detected++
				}

				flp = flpc.LandmarkPoint(leftEye, rightEye, imgParams, perturb, true)
				if flp != nil {
					detected++
				}
			}
		}
		for _, mouth := range mouthCascades {
			for _, flpc := range flpcs[mouth] {
				flp := flpc.LandmarkPoint(leftEye, rightEye, imgParams, perturb, false)
				if flp != nil {
					detected++
				}
			}
		}

		flp := flpcs["lp84"][0].LandmarkPoint(leftEye, rightEye, imgParams, perturb, true)
		if flp != nil {
			detected++
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

	flploc := &pigo.Puploc{Row: 10, Col: 10, Scale: 20, Perturbs: perturb}

	// For benchmarking we are using common values for left and right eye.
	puploc := plc.RunClassifier(flploc, imgParams, 0.0, false)

	runtime.GC()
	b.ResetTimer()

	for b.Loop() {
		plc.LandmarkPoint(puploc, puploc, imgParams, perturb, false)
	}
}

func BenchmarkFlploc_Detection(b *testing.B) {
	classifier = pigo.UnpackCascade()
	plc, err = pigo.UnpackPlc()
	assert.Nil(b, err)

	cParams.Pixels = pixs

	runtime.GC()
	b.ResetTimer()

	dets := classifier.RunClassifier(cParams, 0.0)
	dets = classifier.ClusterDetections(dets, 0.1)

	for b.Loop() {
		for _, det := range dets {
			// left eye
			puploc := &pigo.Puploc{
				Row:      det.Row - int(0.075*float32(det.Scale)),
				Col:      det.Col - int(0.175*float32(det.Scale)),
				Scale:    float32(det.Scale) * 0.25,
				Perturbs: perturb,
			}
			leftEye := plc.RunClassifier(puploc, imgParams, 0.0, false)

			// right eye
			puploc = &pigo.Puploc{
				Row:      det.Row - int(0.075*float32(det.Scale)),
				Col:      det.Col + int(0.185*float32(det.Scale)),
				Scale:    float32(det.Scale) * 0.25,
				Perturbs: perturb,
			}
			rightEye := plc.RunClassifier(puploc, imgParams, 0.0, false)

			plc.LandmarkPoint(leftEye, rightEye, imgParams, perturb, false)

		}
	}
	_ = dets
}
