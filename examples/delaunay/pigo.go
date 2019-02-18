package main

import "C"

import (
	"image"
	"image/color"
	"io/ioutil"
	"log"
	"runtime"
	"unsafe"

	"github.com/esimov/pigo/core"
	"github.com/esimov/triangle"
)

var (
	cascade    []byte
	err        error
	p          *pigo.Pigo
	classifier *pigo.Pigo
)

type SubImager interface {
	SubImage(r image.Rectangle) image.Image
}

type pixs struct {
	rows, cols int
}

func main() {}

//export FindFaces
func FindFaces(pixels []uint8) uintptr {
	px := &pixs{
		rows: 480,
		cols: 640,
	}

	proc := &triangle.Processor{
		BlurRadius:      4,
		SobelThreshold:  10,
		PointsThreshold: 20,
		MaxPoints:       100,
		Wireframe:       0,
		Noise:           0,
		StrokeWidth:     1,
		IsSolid:         false,
		Grayscale:       false,
		OutputToSVG:     false,
		OutputInWeb:     false,
	}
	tri := &triangle.Image{*proc}

	pointCh := make(chan uintptr)

	dets := px.clusterDetection(pixels)
	img := px.pixToImage(pixels)

	tFaces := make([][]int, len(dets))
	result := make([][]int, len(dets))
	det := make([]int, 0, len(dets))

	totalPixDim := 0

	go func() {
		for i := 0; i < len(dets); i++ {
			if dets[i].Q >= 5.0 {
				result[i] = append(result[i], dets[i].Row, dets[i].Col, dets[i].Scale)
				rect := image.Rect(
					dets[i].Col-dets[i].Scale/2,
					dets[i].Row-dets[i].Scale/2,
					dets[i].Scale,
					dets[i].Scale,
				)

				subImg := img.(SubImager).SubImage(rect)
				bounds := subImg.Bounds()

				if bounds.Dx() > 1 && bounds.Dy() > 1 {
					triRes, _, _, err := tri.Draw(subImg, false, func() {})
					if err != nil {
						log.Fatal(err.Error())
					}
					triPix := px.imgToPix(triRes)
					tFaces[i] = append(tFaces[i], triPix...)

					// Prepend the top left coordinates of the detected faces to the delaunay triangles.
					tFaces[i] = append([]int{len(triPix), bounds.Min.X, bounds.Min.Y}, tFaces[i]...)
					totalPixDim += len(triPix)
				}
			}
		}

		// Since in Go we cannot transfer a 2d array trough an array pointer
		// we have to transform it into 1d array.
		for _, v := range result {
			det = append(det, v...)
		}

		// Convert the multidimmensional slice containing the triangulated images to 1d slice.
		convTri := make([]int, 0, len(result)+totalPixDim)
		for _, v := range tFaces {
			convTri = append(convTri, v...)
		}

		// Include as a first slice element the number of detected faces.
		// We need to transfer this value in order to define the Python array buffer length.
		det = append([]int{len(result), 0, 0}, det...)

		// Append the generated triangle slices to detected faces array.
		det = append(det, convTri...)

		// Convert the slice into an array pointer.
		s := *(*[]int)(unsafe.Pointer(&det))
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
func (px pixs) clusterDetection(pixels []uint8) []pigo.Detection {
	cParams := pigo.CascadeParams{
		MinSize:     20,
		MaxSize:     1000,
		ShiftFactor: 0.15,
		ScaleFactor: 1.1,
		ImageParams: pigo.ImageParams{
			Pixels: pixels,
			Rows:   px.rows,
			Cols:   px.cols,
			Dim:    px.cols,
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

// pixToImage converts the pixel array to an image.
func (px pixs) pixToImage(pixels []uint8) image.Image {
	width, height := px.cols, px.rows
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	c := color.RGBA{
		R: uint8(0),
		G: uint8(0),
		B: uint8(0),
		A: uint8(255),
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c.B = uint8(pixels[y*3+x])
			c.G = uint8(pixels[y*3+x+1])
			c.R = uint8(pixels[y*3+x+2])

			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// imgToPix converts the image to a pixel array.
func (px pixs) imgToPix(img image.Image) []int {
	bounds := img.Bounds()
	x := bounds.Dx()
	y := bounds.Dy()
	pixels := make([]int, 0, x*y*3)

	for i := bounds.Min.X; i < bounds.Max.X; i++ {
		for j := bounds.Min.Y; j < bounds.Max.Y; j++ {
			r, g, b, _ := img.At(i, j).RGBA()
			pixels = append(pixels, int(r>>8), int(g>>8), int(b>>8), 255)
		}
	}
	return pixels
}
