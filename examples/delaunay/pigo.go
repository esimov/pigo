package main

import "C"

import (
	"image"
	"image/color"
	"io/ioutil"
	"log"
	"runtime"
	"unsafe"

	pigo "github.com/esimov/pigo/core"
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
		BlurRadius:      1,
		SobelThreshold:  2,
		PointsThreshold: 2,
		MaxPoints:       200,
		Wireframe:       0,
		Noise:           0,
		StrokeWidth:     1,
		IsSolid:         true,
		Grayscale:       false,
		OutputToSVG:     false,
		OutputInWeb:     false,
	}
	tri := &triangle.Image{*proc}

	pointCh := make(chan uintptr)
	go func() {
		img := px.pixToImage(pixels)
		grayscale := pigo.RgbToGrayscale(img.(*image.NRGBA))

		dets := px.clusterDetection(grayscale)
		tFaces := make([][]int, len(dets))
		totalPixDim := 0

		for i := 0; i < len(dets); i++ {
			if dets[i].Q >= 5.0 {
				rect := image.Rect(
					dets[i].Col-dets[i].Scale/2,
					dets[i].Row-dets[i].Scale/2,
					dets[i].Col+dets[i].Scale/2,
					dets[i].Row+dets[i].Scale/2,
				)
				subImg := img.(SubImager).SubImage(rect)
				bounds := subImg.Bounds()

				if bounds.Dx() > 1 && bounds.Dy() > 1 {
					res, _, _, err := tri.Draw(subImg, nil, func() {})
					if err != nil {
						log.Fatal(err.Error())
					}
					triPix := px.imgToPix(res)
					tFaces[i] = append(tFaces[i], triPix...)

					// Prepend the box size and the top left coordinates of the detected faces to the delaunay triangles.
					tFaces[i] = append([]int{
						len(triPix),
						dets[i].Col - dets[i].Scale/2,
						dets[i].Row - dets[i].Scale/2,
						dets[i].Scale,
					}, tFaces[i]...)

					totalPixDim += len(triPix)
				}
			}
		}
		result := make([]int, 0, len(dets))

		// Convert the multidimensional slice containing the triangulated images to 1d slice.
		convTri := make([]int, 0, len(result)*totalPixDim)
		for _, face := range tFaces {
			convTri = append(convTri, face...)
		}
		// Include as a first slice element the number of detected faces.
		// We need to transfer this value in order to define the Python array buffer length.
		result = append([]int{len(dets)}, result...)

		// Append the generated triangle slices to the detected faces array.
		result = append(result, convTri...)

		// Convert the slice into an array pointer.
		s := *(*[]uint8)(unsafe.Pointer(&result))
		p := uintptr(unsafe.Pointer(&s[0]))

		// Ensure `result` is not freed up by GC prematurely.
		runtime.KeepAlive(result)

		pointCh <- p
	}()
	// return the pointer address
	return <-pointCh
}

// clusterDetection runs Pigo face detector core methods
// and returns a cluster with the detected faces coordinates.
func (px pixs) clusterDetection(pixels []uint8) []pigo.Detection {
	cParams := pigo.CascadeParams{
		MinSize:     100,
		MaxSize:     600,
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
		cascade, err = ioutil.ReadFile("../../cascade/facefinder")
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
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	c := color.NRGBA{
		R: uint8(0),
		G: uint8(0),
		B: uint8(0),
		A: uint8(255),
	}

	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X*3; x += 3 {
			c.R = uint8(pixels[x+y*width*3])
			c.G = uint8(pixels[x+y*width*3+1])
			c.B = uint8(pixels[x+y*width*3+2])

			img.SetNRGBA(int(x/3), y, c)
		}
	}
	return img
}

// imgToPix converts the image to a pixel array.
func (px pixs) imgToPix(img image.Image) []int {
	bounds := img.Bounds()
	pixels := make([]int, 0, bounds.Max.X*bounds.Max.Y*3)
	rs := make([]int, 0, bounds.Max.X*bounds.Max.Y)
	gs := make([]int, 0, bounds.Max.X*bounds.Max.Y)
	bs := make([]int, 0, bounds.Max.X*bounds.Max.Y)

	for i := bounds.Min.X; i < bounds.Max.X; i++ {
		for j := bounds.Min.Y; j < bounds.Max.Y; j++ {
			r, g, b, _ := img.At(i, j).RGBA()
			rs = append(rs, int(r>>8))
			gs = append(gs, int(g>>8))
			bs = append(bs, int(b>>8))
		}
	}
	pixels = append(append(append(append(pixels, rs...), gs...), bs...))
	return pixels
}
