package main

import "C"

import (
	"log"
	"os"
	"runtime"
	"unsafe"

	pigo "github.com/esimov/pigo/core"
)

var (
	cascade          []byte
	puplocCascade    []byte
	faceClassifier   *pigo.Pigo
	puplocClassifier *pigo.PuplocCascade
	imageParams      *pigo.ImageParams
	err              error
)

func main() {}

//export FindFaces
func FindFaces(pixels []uint8) uintptr {
	pointCh := make(chan uintptr)

	results := clusterDetection(pixels, 480, 640)
	dets := make([][]int, len(results))

	for i := 0; i < len(results); i++ {
		dets[i] = append(dets[i], results[i].Row, results[i].Col, results[i].Scale, int(results[i].Q), 1)
		// left eye
		puploc := &pigo.Puploc{
			Row:      results[i].Row - int(0.085*float32(results[i].Scale)),
			Col:      results[i].Col - int(0.185*float32(results[i].Scale)),
			Scale:    float32(results[i].Scale) * 0.4,
			Perturbs: 50,
		}
		det := puplocClassifier.RunDetector(*puploc, *imageParams, 0.0, false)
		if det.Row > 0 && det.Col > 0 {
			dets[i] = append(dets[i], det.Row, det.Col, int(det.Scale), int(results[i].Q), 0)
		}

		// right eye
		puploc = &pigo.Puploc{
			Row:      results[i].Row - int(0.085*float32(results[i].Scale)),
			Col:      results[i].Col + int(0.185*float32(results[i].Scale)),
			Scale:    float32(results[i].Scale) * 0.4,
			Perturbs: 50,
		}

		det = puplocClassifier.RunDetector(*puploc, *imageParams, 0.0, false)
		if det.Row > 0 && det.Col > 0 {
			dets[i] = append(dets[i], det.Row, det.Col, int(det.Scale), int(results[i].Q), 0)
		}
	}

	coords := make([]int, 0, len(dets))
	go func() {
		// Since in Go we cannot transfer a 2d array through an array pointer
		// we have to transform it into 1d array.
		for _, v := range dets {
			coords = append(coords, v...)
		}

		// Include as a first slice element the number of detected faces.
		// We need to transfer this value in order to define the Python array buffer length.
		coords = append([]int{len(dets), 0, 0, 0, 0}, coords...)

		// Convert the slice into an array pointer.
		s := *(*[]uint8)(unsafe.Pointer(&coords))
		p := uintptr(unsafe.Pointer(&s[0]))

		// Ensure `det` is not freed up by GC prematurely.
		runtime.KeepAlive(coords)

		// return the pointer address
		pointCh <- p
	}()
	return <-pointCh
}

// clusterDetection runs Pigo face detector core methods
// and returns a cluster with the detected faces coordinates.
func clusterDetection(pixels []uint8, rows, cols int) []pigo.Detection {
	imageParams = &pigo.ImageParams{
		Pixels: pixels,
		Rows:   rows,
		Cols:   cols,
		Dim:    cols,
	}
	cParams := pigo.CascadeParams{
		MinSize:     260,
		MaxSize:     640,
		ShiftFactor: 0.1,
		ScaleFactor: 1.1,
		ImageParams: *imageParams,
	}

	// Ensure that the face detection classifier is loaded only once.
	if len(cascade) == 0 {
		cascade, err = os.ReadFile("../../cascade/facefinder")
		if err != nil {
			log.Fatalf("Error reading the cascade file: %v", err)
		}
		p := pigo.NewPigo()

		// Unpack the binary file. This will return the number of cascade trees,
		// the tree depth, the threshold and the prediction from tree's leaf nodes.
		faceClassifier, err = p.Unpack(cascade)
		if err != nil {
			log.Fatalf("Error unpacking the cascade file: %s", err)
		}
	}

	// Ensure that we load the pupil localization cascade only once
	if len(puplocCascade) == 0 {
		puplocCascade, err := os.ReadFile("../../cascade/puploc")
		if err != nil {
			log.Fatalf("Error reading the puploc cascade file: %s", err)
		}
		puplocClassifier, err = puplocClassifier.UnpackCascade(puplocCascade)
		if err != nil {
			log.Fatalf("Error unpacking the puploc cascade file: %s", err)
		}
	}

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	dets := faceClassifier.RunCascade(cParams, 0.0)

	// Calculate the intersection over union (IoU) of two clusters.
	dets = faceClassifier.ClusterDetections(dets, 0.0)

	return dets
}
