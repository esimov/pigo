package main

import "C"

import (
	"fmt"
	"io/ioutil"
	"log"
	"reflect"
	"runtime"
	"unsafe"

	"github.com/esimov/pigo/core"
)

var (
	cascade    []byte
	err        error
	p          *pigo.Pigo
	classifier *pigo.Pigo
)

func main() {}

//export FindFaces
func FindFaces(pixels []uint8) uintptr {
	if len(pixels) > 0 {
		dets := clusterDetection(pixels, 480, 640)
		result := make([][]int, len(dets))

		for i := 0; i < len(dets); i++ {
			if dets[i].Q >= 5.0 {
				result[i] = append(result[i], dets[i].Row, dets[i].Col, dets[i].Scale)
			}
		}
		//fmt.Println(dets)
		fmt.Println(result)

		if len(result) > 0 {
			det := make([]int, 0, len(result))
			for _, v := range result {
				det = append(det, v...)
			}
			det = append([]int{len(result), 0, 0}, det...)
			fmt.Println(det)

			s := *(*[]int)(unsafe.Pointer(&det))
			p := uintptr(unsafe.Pointer(&s[0]))
			return p

			sh := &reflect.SliceHeader{
				Data: p,
				Len:  len(result),
				Cap:  len(result),
			}

			data := *(*[][]int)(unsafe.Pointer(sh))

			fmt.Println(data)

			runtime.KeepAlive(result)
			return uintptr(unsafe.Pointer(&data[0]))
		}
	}
	return 0
}

func clusterDetection(pixels []uint8, rows, cols int) []pigo.Detection {
	// cfp := *(*[]byte)(unsafe.Pointer(&cascadeFile))
	// p := uintptr(unsafe.Pointer(&cfp[0]))

	// size := len(cascadeFile)
	// var data []byte

	// sh := (*reflect.SliceHeader)(unsafe.Pointer(&data))
	// sh.Data = p
	// sh.Len = size
	// sh.Cap = size

	// fmt.Println(cascadeFile)
	// fmt.Println(data)
	// fmt.Println(uintptr(unsafe.Pointer(&cfp[0])))
	fmt.Println("P:", len(pixels))
	fmt.Println("DIM:", rows, cols)

	cParams := pigo.CascadeParams{
		MinSize:     20,
		MaxSize:     1000,
		ShiftFactor: 0.22,
		ScaleFactor: 1.1,
		ImageParams: pigo.ImageParams{
			Pixels: pixels,
			Rows:   rows,
			Cols:   cols,
			Dim:    cols,
		},
	}

	if len(cascade) == 0 {
		cascade, err = ioutil.ReadFile("../../data/facefinder")
		if err != nil {
			log.Fatalf("Error reading the cascade file: %v", err)
		}

		// Unpack the binary file. This will return the number of cascade trees,
		// the tree depth, the threshold and the prediction from tree's leaf nodes.
		classifier, err = p.Unpack(cascade)
		if err != nil {
			log.Fatalf("Error reading the cascade file: %s", err)
		}
	}

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	dets := classifier.RunCascade(cParams, 0.0)

	// Calculate the intersection over union (IoU) of two clusters.
	dets = classifier.ClusterDetections(dets, 0)

	return dets
}
