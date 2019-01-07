package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/esimov/pigo/core"
	"github.com/fogleman/gg"
)

const banner = `
┌─┐┬┌─┐┌─┐
├─┘││ ┬│ │
┴  ┴└─┘└─┘

Go (Golang) Face detection library.
    Version: %s

`

// Version indicates the current build version.
var Version string

var (
	// Flags
	source       = flag.String("in", "", "Source image")
	destination  = flag.String("out", "", "Destination image")
	cascadeFile  = flag.String("cf", "", "Cascade binary file")
	minSize      = flag.Int("min", 20, "Minimum size of face")
	maxSize      = flag.Int("max", 1000, "Maximum size of face")
	shiftFactor  = flag.Float64("shift", 0.1, "Shift detection window by percentage")
	scaleFactor  = flag.Float64("scale", 1.1, "Scale detection window by percentage")
	angle        = flag.Float64("angle", 0.0, "0.0 is 0 radians and 1.0 is 2*pi radians")
	iouThreshold = flag.Float64("iou", 0.2, "Intersection over union (IoU) threshold")
	circleMarker = flag.Bool("circle", false, "Use circle as detection marker")
	outputAsJSON = flag.Bool("json", false, "Output face box coordinates into a json file")
)

var dc *gg.Context

// faceDetector struct contains Pigo face detector general settings.
type faceDetector struct {
	cascadeFile  string
	minSize      int
	maxSize      int
	shiftFactor  float64
	scaleFactor  float64
	iouThreshold float64
}

// detectionResult contains the coordinates of the detected faces and the base64 converted image.
type detectionResult struct {
	rects []image.Rectangle
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, fmt.Sprintf(banner, Version))
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(*source) == 0 || len(*destination) == 0 || len(*cascadeFile) == 0 {
		log.Fatal("Usage: pigo -in input.jpg -out out.png -cf data/facefinder")
	}

	fileTypes := []string{".jpg", ".jpeg", ".png"}
	ext := filepath.Ext(*destination)

	if !inSlice(ext, fileTypes) {
		log.Fatalf("Output file type not supported: %v", ext)
	}

	if *scaleFactor < 1 {
		log.Fatal("Scale factor must be greater than 1.")
	}

	// Progress indicator
	s := new(spinner)
	s.start("Processing...")
	start := time.Now()

	fd := newFaceDetector(*cascadeFile, *minSize, *maxSize, *shiftFactor, *scaleFactor, *iouThreshold)
	faces, err := fd.detectFaces(*source)
	if err != nil {
		log.Fatalf("Detection error: %v", err)
	}

	_, rects, err := fd.drawFaces(faces, *circleMarker)
	if err != nil {
		log.Fatalf("Error creating the image output: %s", err)
	}

	resp := detectionResult{
		rects: rects,
	}

	out, err := json.Marshal(resp)
	if *outputAsJSON {
		ioutil.WriteFile("output.json", out, 0644)
	}

	s.stop()
	fmt.Printf("\nDone in: \x1b[92m%.2fs\n", time.Since(start).Seconds())
}

// newFaceDetector initialises the constructor function.
func newFaceDetector(cf string, minSize, maxSize int, shf, scf, iou float64) *faceDetector {
	return &faceDetector{
		cascadeFile:  cf,
		minSize:      minSize,
		maxSize:      maxSize,
		shiftFactor:  shf,
		scaleFactor:  scf,
		iouThreshold: iou,
	}
}

// detectFaces run the detection algorithm over the provided source image.
func (fd *faceDetector) detectFaces(source string) ([]pigo.Detection, error) {
	src, err := pigo.GetImage(source)
	if err != nil {
		return nil, err
	}

	pixels := pigo.RgbToGrayscale(src)
	cols, rows := src.Bounds().Max.X, src.Bounds().Max.Y

	dc = gg.NewContext(cols, rows)
	dc.DrawImage(src, 0, 0)

	cParams := pigo.CascadeParams{
		MinSize:     fd.minSize,
		MaxSize:     fd.maxSize,
		ShiftFactor: fd.shiftFactor,
		ScaleFactor: fd.scaleFactor,
		ImageParams: pigo.ImageParams{
			Pixels: pixels,
			Rows:   rows,
			Cols:   cols,
			Dim:    cols,
		},
	}

	cascadeFile, err := ioutil.ReadFile(fd.cascadeFile)
	if err != nil {
		return nil, err
	}

	pigo := pigo.NewPigo()
	// Unpack the binary file. This will return the number of cascade trees,
	// the tree depth, the threshold and the prediction from tree's leaf nodes.
	classifier, err := pigo.Unpack(cascadeFile)
	if err != nil {
		return nil, err
	}

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	faces := classifier.RunCascade(cParams, *angle)

	// Calculate the intersection over union (IoU) of two clusters.
	faces = classifier.ClusterDetections(faces, fd.iouThreshold)

	return faces, nil
}

// drawFaces marks the detected faces with a circle in case isCircle is true, otherwise marks with a rectangle.
func (fd *faceDetector) drawFaces(faces []pigo.Detection, isCircle bool) ([]byte, []image.Rectangle, error) {
	var (
		qThresh float32 = 5.0
		rects   []image.Rectangle
	)

	for _, face := range faces {
		if face.Q > qThresh {
			if isCircle {
				dc.DrawArc(
					float64(face.Col),
					float64(face.Row),
					float64(face.Scale/2),
					0,
					2*math.Pi,
				)
			} else {
				dc.DrawRectangle(
					float64(face.Col-face.Scale/2),
					float64(face.Row-face.Scale/2),
					float64(face.Scale),
					float64(face.Scale),
				)
			}
			rects = append(rects, image.Rect(
				face.Col-face.Scale/2,
				face.Row-face.Scale/2,
				face.Scale,
				face.Scale,
			))
			dc.SetLineWidth(2.0)
			dc.SetStrokeStyle(gg.NewSolidPattern(color.RGBA{R: 255, G: 0, B: 0, A: 255}))
			dc.Stroke()
		}
	}

	img := dc.Image()
	output, err := os.OpenFile(*destination, os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		return nil, nil, err
	}
	ext := filepath.Ext(output.Name())

	switch ext {
	case ".jpg", ".jpeg":
		jpeg.Encode(output, img, &jpeg.Options{Quality: 100})
	case ".png":
		png.Encode(output, img)
	}
	rf, err := ioutil.ReadFile(*destination)

	return rf, rects, err
}

type spinner struct {
	stopChan chan struct{}
}

// Start process
func (s *spinner) start(message string) {
	s.stopChan = make(chan struct{}, 1)

	go func() {
		for {
			for _, r := range `-\|/` {
				select {
				case <-s.stopChan:
					return
				default:
					fmt.Printf("\r%s%s %c%s", message, "\x1b[92m", r, "\x1b[39m")
					time.Sleep(time.Millisecond * 100)
				}
			}
		}
	}()
}

// End process
func (s *spinner) stop() {
	s.stopChan <- struct{}{}
}

// inSlice check if a slice contains the string value.
func inSlice(ext string, types []string) bool {
	for _, t := range types {
		if t == ext {
			return true
		}
	}
	return false
}
