package detector

import (
	"fmt"

	pigo "github.com/esimov/pigo/pkg"
)

const perturb = 63

var (
	faceClassifier   *pigo.Pigo
	puplocClassifier *pigo.PuplocCascade
	flpcs            map[string][]*pigo.FlpCascade
	imgParams        *pigo.ImageParams
	err              error
)

var (
	eyeCascades  = []string{"lp46", "lp44", "lp42", "lp38", "lp312"}
	mouthCascade = []string{"lp93", "lp84", "lp82", "lp81"}
)

// UnpackCascades unpack all of the existing cascade files.
func (d *Detector) UnpackCascades() error {
	p := pigo.NewPigo()

	// Unpack the binary file. This will return the number of cascade trees,
	// the tree depth, the threshold and the prediction from tree's leaf nodes.
	faceClassifier = p.Unpack()
	plc := pigo.NewPuplocCascade()

	puplocClassifier, err = plc.ReadCascade()
	if err != nil {
		return fmt.Errorf("error unpacking the pupil localization cascade file: %w", err)
	}

	flpcs, err = plc.ReadCascadeDir()
	if err != nil {
		return fmt.Errorf("failed to unpack the facial landmark points cascade files: %w", err)
	}

	return nil
}

// DetectFaces runs the cluster detection over the webcam frame
// received as a pixel array and returns the detected faces.
func (d *Detector) DetectFaces(pixels []uint8, width, height int) [][]int {
	results := d.clusterDetection(pixels, width, height)
	dets := make([][]int, len(results))

	for i := 0; i < len(results); i++ {
		dets[i] = append(dets[i], results[i].Row, results[i].Col, results[i].Scale, int(results[i].Q))
	}

	return dets
}

// DetectLeftPupil detects the left pupil
func (d *Detector) DetectLeftPupil(results []int) *pigo.Puploc {
	puploc := &pigo.Puploc{
		Row:      results[0] - int(0.085*float32(results[2])),
		Col:      results[1] - int(0.185*float32(results[2])),
		Scale:    float32(results[2]) * 0.4,
		Perturbs: perturb,
	}

	leftEye := puplocClassifier.RunDetector(puploc, *imgParams, 0.0, false)
	if leftEye.Row > 0 && leftEye.Col > 0 {
		return leftEye
	}

	return nil
}

// DetectRightPupil detects the right pupil
func (d *Detector) DetectRightPupil(results []int) *pigo.Puploc {
	puploc := &pigo.Puploc{
		Row:      results[0] - int(0.085*float32(results[2])),
		Col:      results[1] + int(0.185*float32(results[2])),
		Scale:    float32(results[2]) * 0.4,
		Perturbs: perturb,
	}

	rightEye := puplocClassifier.RunDetector(puploc, *imgParams, 0.0, false)
	if rightEye.Row > 0 && rightEye.Col > 0 {
		return rightEye
	}

	return nil
}

// DetectLandmarkPoints detects the landmark points
func (d *Detector) DetectLandmarkPoints(leftEye, rightEye *pigo.Puploc) [][]int {
	var (
		det = make([][]int, 15)
		idx int
	)

	for _, eye := range eyeCascades {
		for _, flpc := range flpcs[eye] {
			flp := flpc.LandmarkPoint(leftEye, rightEye, *imgParams, perturb, false)
			if flp.Row > 0 && flp.Col > 0 {
				det[idx] = append(det[idx], flp.Col, flp.Row, int(flp.Scale))
			}
			idx++

			flp = flpc.LandmarkPoint(leftEye, rightEye, *imgParams, perturb, true)
			if flp.Row > 0 && flp.Col > 0 {
				det[idx] = append(det[idx], flp.Col, flp.Row, int(flp.Scale))
			}
			idx++
		}
	}

	for _, mouth := range mouthCascade {
		for _, flpc := range flpcs[mouth] {
			flp := flpc.LandmarkPoint(leftEye, rightEye, *imgParams, perturb, false)
			if flp.Row > 0 && flp.Col > 0 {
				det[idx] = append(det[idx], flp.Col, flp.Row, int(flp.Scale))
			}
			idx++
		}
	}

	flp := flpcs["lp84"][0].LandmarkPoint(leftEye, rightEye, *imgParams, perturb, true)
	if flp.Row > 0 && flp.Col > 0 {
		det[idx] = append(det[idx], flp.Col, flp.Row, int(flp.Scale))
	}

	return det
}

// clusterDetection runs Pigo face detector core methods
// and returns a cluster with the detected faces coordinates.
func (d *Detector) clusterDetection(pixels []uint8, width, height int) []pigo.Detection {
	imgParams = &pigo.ImageParams{
		Pixels: pixels,
		Rows:   width,
		Cols:   height,
		Dim:    height,
	}
	cParams := pigo.CascadeParams{
		MinSize:     200,
		MaxSize:     480,
		ShiftFactor: 0.1,
		ScaleFactor: 1.1,
		ImageParams: *imgParams,
	}

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	dets := faceClassifier.RunCascade(cParams, 0.0)

	// Calculate the intersection over union (IoU) of two clusters.
	dets = faceClassifier.ClusterDetections(dets, 0.1)

	return dets
}
