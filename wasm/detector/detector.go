package detector

import (
	"errors"
	"fmt"

	pigo "github.com/esimov/pigo/core"
)

// FlpCascade holds the binary representation of the facial landmark points cascade files
type FlpCascade struct {
	*pigo.PuplocCascade
	error
}

var (
	cascade          []byte
	puplocCascade    []byte
	faceClassifier   *pigo.Pigo
	puplocClassifier *pigo.PuplocCascade
	flpcs            map[string][]*FlpCascade
	imgParams        *pigo.ImageParams
	err              error
)

var (
	eyeCascades  = []string{"lp46", "lp44", "lp42", "lp38", "lp312"}
	mouthCascade = []string{"lp93", "lp84", "lp82", "lp81"}
)

// UnpackCascades unpack all of used cascade files.
func (d *Detector) UnpackCascades() error {
	p := pigo.NewPigo()

	cascade, err = d.FetchCascade("https://raw.githubusercontent.com/esimov/pigo/master/cascade/facefinder")
	if err != nil {
		return errors.New("error reading the facefinder cascade file")
	}
	// Unpack the binary file. This will return the number of cascade trees,
	// the tree depth, the threshold and the prediction from tree's leaf nodes.
	faceClassifier, err = p.Unpack(cascade)
	if err != nil {
		return errors.New("error unpacking the facefinder cascade file")
	}

	plc := pigo.NewPuplocCascade()

	puplocCascade, err = d.FetchCascade("https://raw.githubusercontent.com/esimov/pigo/master/cascade/puploc")
	if err != nil {
		return errors.New("error reading the puploc cascade file")
	}

	puplocClassifier, err = plc.UnpackCascade(puplocCascade)
	if err != nil {
		return errors.New("error unpacking the puploc cascade file")
	}

	flpcs, err = d.parseCascadeFiles("https://raw.githubusercontent.com/esimov/pigo/master/cascade/lps/")
	if err != nil {
		return errors.New("error unpacking the facial landmark points detection cascades")
	}
	return nil
}

// DetectFaces runs the cluster detection over the frame received as a pixel array.
func (d *Detector) DetectFaces(pixels []uint8, width, height int) [][]int {
	results := d.clusterDetection(pixels, width, height)
	fmt.Println(results)
	dets := make([][]int, len(results))

	for i := 0; i < len(results); i++ {
		dets[i] = append(dets[i], results[i].Row, results[i].Col, results[i].Scale, int(results[i].Q))
		// left eye
		puploc := &pigo.Puploc{
			Row:      results[i].Row - int(0.085*float32(results[i].Scale)),
			Col:      results[i].Col - int(0.185*float32(results[i].Scale)),
			Scale:    float32(results[i].Scale) * 0.4,
			Perturbs: 63,
		}
		leftEye := puplocClassifier.RunDetector(*puploc, *imgParams, 0.0, false)
		if leftEye.Row > 0 && leftEye.Col > 0 {
			dets[i] = append(dets[i], leftEye.Row, leftEye.Col, int(leftEye.Scale), int(results[i].Q))
		}

		// right eye
		puploc = &pigo.Puploc{
			Row:      results[i].Row - int(0.085*float32(results[i].Scale)),
			Col:      results[i].Col + int(0.185*float32(results[i].Scale)),
			Scale:    float32(results[i].Scale) * 0.4,
			Perturbs: 63,
		}

		rightEye := puplocClassifier.RunDetector(*puploc, *imgParams, 0.0, false)
		if rightEye.Row > 0 && rightEye.Col > 0 {
			dets[i] = append(dets[i], rightEye.Row, rightEye.Col, int(rightEye.Scale), int(results[i].Q))
		}

		// Traverse all the eye cascades and run the detector on each of them.
		for _, eye := range eyeCascades {
			for _, flpc := range flpcs[eye] {
				flp := flpc.FindLandmarkPoints(leftEye, rightEye, *imgParams, puploc.Perturbs, false)
				if flp.Row > 0 && flp.Col > 0 {
					dets[i] = append(dets[i], flp.Row, flp.Col, int(flp.Scale), int(results[i].Q))
				}

				flp = flpc.FindLandmarkPoints(leftEye, rightEye, *imgParams, puploc.Perturbs, true)
				if flp.Row > 0 && flp.Col > 0 {
					dets[i] = append(dets[i], flp.Row, flp.Col, int(flp.Scale), int(results[i].Q))
				}
			}
		}

		// Traverse all the mouth cascades and run the detector on each of them.
		for _, mouth := range mouthCascade {
			for _, flpc := range flpcs[mouth] {
				flp := flpc.FindLandmarkPoints(leftEye, rightEye, *imgParams, puploc.Perturbs, false)
				if flp.Row > 0 && flp.Col > 0 {
					dets[i] = append(dets[i], flp.Row, flp.Col, int(flp.Scale), int(results[i].Q))
				}
			}
		}
		flp := flpcs["lp84"][0].FindLandmarkPoints(leftEye, rightEye, *imgParams, puploc.Perturbs, true)
		if flp.Row > 0 && flp.Col > 0 {
			dets[i] = append(dets[i], flp.Row, flp.Col, int(flp.Scale), int(results[i].Q))
		}
	}
	return dets
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
		MinSize:     100,
		MaxSize:     1200,
		ShiftFactor: 0.1,
		ScaleFactor: 1.1,
		ImageParams: *imgParams,
	}

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	dets := faceClassifier.RunCascade(cParams, 0.0)

	// Calculate the intersection over union (IoU) of two clusters.
	dets = faceClassifier.ClusterDetections(dets, 0.0)

	return dets
}

// parseCascadeFiles reads the facial landmark points cascades from the provided url.
func (d *Detector) parseCascadeFiles(path string) (map[string][]*FlpCascade, error) {
	cascades := append(eyeCascades, mouthCascade...)
	flpcs := make(map[string][]*FlpCascade)

	pl := pigo.NewPuplocCascade()

	for _, cascade := range cascades {
		puplocCascade, err = d.FetchCascade(path + cascade)
		if err != nil {
			d.Log("Error reading the cascade file: %v", err)
		}
		flpc, err := pl.UnpackCascade(puplocCascade)
		flpcs[cascade] = append(flpcs[cascade], &FlpCascade{flpc, err})
	}
	return flpcs, err
}
