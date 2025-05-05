package pigo_test

import (
	"log"
	"os"
	"runtime"
	"testing"

	pigo "github.com/esimov/pigo/core"
)

var flpc []byte

const perturb = 63

func init() {
	flpc, err = os.ReadFile("../cascade/lps/lp42")
	if err != nil {
		log.Fatalf("missing cascade file: %v", err)
	}
}

func TestFlploc_UnpackCascadeFileShouldNotBeNil(t *testing.T) {
	plc, err = pl.UnpackCascade(flpc)
	if err != nil {
		t.Fatalf("failed unpacking the cascade file: %v", err)
	}
}

func TestFlploc_LandmarkDetectorShouldReturnDetectionPoints(t *testing.T) {
	// Unpack the binary file. This will return the number of cascade trees,
	// the tree depth, the threshold and the prediction from tree's leaf nodes.
	p, err = p.Unpack(faceCasc)
	if err != nil {
		t.Fatalf("error reading the cascade file: %s", err)
	}

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	dets := p.RunCascade(*cParams, 0.0)
	// Calculate the intersection over union (IoU) of two clusters.
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
			leftEye := plc.RunDetector(*puploc, *imgParams, 0.0, false)

			// right eye
			puploc = &pigo.Puploc{
				Row:      det.Row - int(0.075*float32(det.Scale)),
				Col:      det.Col + int(0.185*float32(det.Scale)),
				Scale:    float32(det.Scale) * 0.25,
				Perturbs: 50,
			}
			rightEye := plc.RunDetector(*puploc, *imgParams, 0.0, false)

			flp := plc.GetLandmarkPoint(leftEye, rightEye, *imgParams, perturb, false)
			landMarkPoints = append(landMarkPoints, *flp)
		}
	}
	if len(landMarkPoints) == 0 {
		t.Fatal("should have been detected facial landmark points")
	}
}

func TestFlploc_LandmarkDetectorShouldReturnCorrectDetectionPoints(t *testing.T) {
	var (
		eyeCascades   = []string{"lp46", "lp44", "lp42", "lp38", "lp312"}
		mouthCascades = []string{"lp93", "lp84", "lp82", "lp81"}
		flpcs         map[string][]*pigo.FlpCascade

		detLandmarkPts int
	)

	// Unpack the binary file. This will return the number of cascade trees,
	// the tree depth, the threshold and the prediction from tree's leaf nodes.
	p, err = p.Unpack(faceCasc)
	if err != nil {
		t.Fatalf("error reading the cascade file: %s", err)
	}

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	dets := p.RunCascade(*cParams, 0.0)
	// Calculate the intersection over union (IoU) of two clusters.
	dets = p.ClusterDetections(dets, 0.1)

	flpcs, err = plc.ReadCascadeDir("../cascade/lps/")
	if err != nil {
		t.Fatalf("error reading the facial landmark points cascade directory: %s", err)
	}

	for _, det := range dets {
		if det.Scale > 50 {
			// left eye
			puploc := &pigo.Puploc{
				Row:      det.Row - int(0.075*float32(det.Scale)),
				Col:      det.Col - int(0.175*float32(det.Scale)),
				Scale:    float32(det.Scale) * 0.25,
				Perturbs: 50,
			}
			leftEye := plc.RunDetector(*puploc, *imgParams, 0.0, false)

			// right eye
			puploc = &pigo.Puploc{
				Row:      det.Row - int(0.075*float32(det.Scale)),
				Col:      det.Col + int(0.185*float32(det.Scale)),
				Scale:    float32(det.Scale) * 0.25,
				Perturbs: 50,
			}
			rightEye := plc.RunDetector(*puploc, *imgParams, 0.0, false)

			for _, eye := range eyeCascades {
				for _, flpc := range flpcs[eye] {
					flp := flpc.GetLandmarkPoint(leftEye, rightEye, *imgParams, perturb, false)
					if flp.Row > 0 && flp.Col > 0 {
						detLandmarkPts++
					}
					flp = flpc.GetLandmarkPoint(leftEye, rightEye, *imgParams, perturb, true)
					if flp.Row > 0 && flp.Col > 0 {
						detLandmarkPts++
					}
				}
			}
			for _, mouth := range mouthCascades {
				for _, flpc := range flpcs[mouth] {
					flp := flpc.GetLandmarkPoint(leftEye, rightEye, *imgParams, perturb, false)
					if flp.Row > 0 && flp.Col > 0 {
						detLandmarkPts++
					}
				}
			}

			flp := flpcs["lp84"][0].GetLandmarkPoint(leftEye, rightEye, *imgParams, perturb, true)
			if flp.Row > 0 && flp.Col > 0 {
				detLandmarkPts++
			}

		}
	}
	expLandmarkPts := 2*len(eyeCascades) + len(mouthCascades) + 1 // landmark points of the left/right eyes, mouth + nose
	if expLandmarkPts != detLandmarkPts {
		t.Fatalf("expected facial landmark points to be detected: %d, got: %d", expLandmarkPts, detLandmarkPts)
	}
}

func BenchmarkFlplocReadCascadeDir(b *testing.B) {
	for i := 0; i < b.N; i++ {
		plc.ReadCascadeDir("../cascade/lps/")
	}
}

func BenchmarkFlplocGetLendmarkPoint(b *testing.B) {
	plc, err = pl.UnpackCascade(puplocCasc)
	if err != nil {
		b.Fatalf("error reading the cascade file: %s", err)
	}

	cParams.Pixels = pixs

	flploc := &pigo.Puploc{Row: 10, Col: 10, Scale: 20, Perturbs: 50}
	// For benchmarking we are using common values for left and right eye.
	puploc := plc.RunDetector(*flploc, *imgParams, 0.0, false)

	runtime.GC()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		plc.GetLandmarkPoint(puploc, puploc, *imgParams, 63, false)
	}
}

func BenchmarkFlplocDetection(b *testing.B) {
	// Unpack the binary file. This will return the number of cascade trees,
	// the tree depth, the threshold and the prediction from tree's leaf nodes.
	p, err := p.Unpack(faceCasc)
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
				leftEye := plc.RunDetector(*puploc, *imgParams, 0.0, false)

				// right eye
				puploc = &pigo.Puploc{
					Row:      det.Row - int(0.075*float32(det.Scale)),
					Col:      det.Col + int(0.185*float32(det.Scale)),
					Scale:    float32(det.Scale) * 0.25,
					Perturbs: 50,
				}
				rightEye := plc.RunDetector(*puploc, *imgParams, 0.0, false)

				plc.GetLandmarkPoint(leftEye, rightEye, *imgParams, 63, false)

			}
		}
	}
	_ = dets
}
