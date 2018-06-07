package main

import (
	"flag"
	"fmt"
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
	iouThreshold = flag.Float64("iou", 0.2, "Intersection over union (IoU) threshold")
	circleMarker = flag.Bool("circle", false, "Use circle as detection marker")
)

var dc *gg.Context

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

	cascadeFile, err := ioutil.ReadFile(*cascadeFile)
	if err != nil {
		log.Fatalf("Error reading the cascade file: %v", err)
	}

	// Progress indicator
	s := new(spinner)
	s.start("Processing...")
	start := time.Now()

	src, err := pigo.GetImage(*source)
	if err != nil {
		log.Fatalf("Cannot open the image file: %v", err)
	}

	sampleImg := pigo.RgbToGrayscale(src)

	cParams := pigo.CascadeParams{
		MinSize:     *minSize,
		MaxSize:     *maxSize,
		ShiftFactor: *shiftFactor,
		ScaleFactor: *scaleFactor,
	}
	cols, rows := src.Bounds().Max.X, src.Bounds().Max.Y
	imgParams := pigo.ImageParams{sampleImg, rows, cols, cols}

	pigo := pigo.NewPigo()
	// Unpack the binary file. This will return the number of cascade trees,
	// the tree depth, the threshold and the prediction from tree's leaf nodes.
	classifier, err := pigo.Unpack(cascadeFile)
	if err != nil {
		log.Fatalf("Error reading the cascade file: %s", err)
	}

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	dets := classifier.RunCascade(imgParams, cParams)

	// Calculate the intersection over union (IoU) of two clusters.
	dets = classifier.ClusterDetections(dets, *iouThreshold)

	dc = gg.NewContext(cols, rows)
	dc.DrawImage(src, 0, 0)

	if err := output(dets, *circleMarker); err != nil {
		log.Fatalf("Cannot save the output image %v", err)
	}

	s.stop()
	fmt.Printf("\nDone in: \x1b[92m%.2fs\n", time.Since(start).Seconds())
}

// output mark the face region with the provided marker (rectangle or circle).
func output(detections []pigo.Detection, isCircle bool) error {
	var qThresh float32 = 5.0

	for i := 0; i < len(detections); i++ {
		if detections[i].Q > qThresh {
			if isCircle {
				dc.DrawArc(
					float64(detections[i].Col),
					float64(detections[i].Row),
					float64(detections[i].Scale/2),
					0,
					2*math.Pi,
				)
			} else {
				dc.DrawRectangle(
					float64(detections[i].Col-detections[i].Scale/2),
					float64(detections[i].Row-detections[i].Scale/2),
					float64(detections[i].Scale),
					float64(detections[i].Scale),
				)
			}
			dc.SetLineWidth(3.0)
			dc.SetStrokeStyle(gg.NewSolidPattern(color.RGBA{R: 255, G: 0, B: 0, A: 255}))
			dc.Stroke()
		}
	}
	img := dc.Image()
	output, err := os.OpenFile(*destination, os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		return err
	}
	ext := filepath.Ext(output.Name())

	switch ext {
	case ".jpg", ".jpeg":
		return jpeg.Encode(output, img, &jpeg.Options{Quality: 100})
	case ".png":
		return png.Encode(output, img)
	}
	return nil
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
