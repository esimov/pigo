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
	cascade    []byte
	err        error
	classifier *pigo.Pigo
)

func main() {}

//export FindFaces
func FindFaces(pixels []uint8) uintptr {
	pointCh := make(chan uintptr)

	dets := clusterDetection(pixels, 480, 640)
	result := make([][]int, len(dets))

	for i := 0; i < len(dets); i++ {
		if dets[i].Q >= 5.0 {
			result[i] = append(result[i], dets[i].Row, dets[i].Col, dets[i].Scale)
		}
	}

	det := make([]int, 0, len(result))
	go func() {
		// Since in Go we cannot transfer a 2d array through an array pointer
		// we have to transform it into 1d array.
		for _, v := range result {
			det = append(det, v...)
		}
		// Include as a first slice element the number of detected faces.
		// We need to transfer this value in order to define the Python array buffer length.
		det = append([]int{len(result), 0, 0}, det...)

		// Convert the slice into an array pointer.
		s := *(*[]uint8)(unsafe.Pointer(&det))
		p := uintptr(unsafe.Pointer(&s[0]))

		// Ensure `det` is not freed up by GC prematurely.
		runtime.KeepAlive(det)

		// return the pointer address
		pointCh <- p
	}()
	return <-pointCh
}

// clusterDetection runs Pigo face detector core methods
// and returns a cluster with the detected faces coordinates.
func clusterDetection(pixels []uint8, rows, cols int) []pigo.Detection {
	cParams := pigo.CascadeParams{
		MinSize:     100,
		MaxSize:     600,
		ShiftFactor: 0.15,
		ScaleFactor: 1.1,
		ImageParams: pigo.ImageParams{
			Pixels: pixels,
			Rows:   rows,
			Cols:   cols,
			Dim:    cols,
		},
	}

	if len(cascade) == 0 {
		cascade, err = os.ReadFile("../../cascade/facefinder")
		if err != nil {
			log.Fatalf("Error reading the cascade file: %s", err)
		}
		p := pigo.NewPigo()

		// Unpack the binary file. This will return the number of cascade trees,
		// the tree depth, the threshold and the prediction from tree's leaf nodes.
		classifier, err = p.Unpack(cascade)
		if err != nil {
			log.Fatalf("Error unpacking the cascade file: %s", err)
		}
	}

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	dets := classifier.RunCascade(cParams, 0.0)

	// Calculate the intersection over union (IoU) of two clusters.
	dets = classifier.ClusterDetections(dets, 0)

	return dets
}
