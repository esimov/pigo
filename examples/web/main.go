package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"

	pigo "github.com/esimov/pigo/core"
	"github.com/fogleman/gg"
)

const boundary = "informs"
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
	shiftFactor  = flag.Float64("shift", 0.15, "Shift detection window by percentage")
	scaleFactor  = flag.Float64("scale", 1.1, "Scale detection window by percentage")
	angle        = flag.Float64("angle", 0.0, "0.0 is 0 radians and 1.0 is 2*pi radians")
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

	if len(*cascadeFile) == 0 {
		log.Fatal("Usage: go run main.go -cf ../../cascade/facefinder")
	}

	if *scaleFactor < 1 {
		log.Fatal("Scale factor must be greater than 1.")
	}

	http.HandleFunc("/cam", webcam)
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func webcam(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary="+boundary)

	cmd := exec.CommandContext(r.Context(), "./capture.py")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("[ERROR] Getting the stdout pipe")
		return
	}
	cmd.Start()

	cascadeFile, err := os.ReadFile(*cascadeFile)
	if err != nil {
		log.Fatalf("[ERROR] reading the cascade file: %v", err)
	}

	p := pigo.NewPigo()
	// Unpack the binary file. This will return the number of cascade trees,
	// the tree depth, the threshold and the prediction from tree's leaf nodes.
	classifier, err := p.Unpack(cascadeFile)
	if err != nil {
		log.Fatalf("[ERROR] reading the cascade file: %v", err)
	}

	mpart := multipart.NewReader(stdout, boundary)
	for {
		p, err := mpart.NextPart()
		if err == io.EOF {
			log.Println("[DEBUG] EOF")
			break
		}
		if err != nil {
			log.Println("[ERROR] reading next part", err)
			return
		}

		data, err := io.ReadAll(p)
		if err != nil {
			log.Println("[ERROR] reading from bytes ", err)
			continue
		}
		img, _, _ := image.Decode(bytes.NewReader(data))

		frameBuffer := new(bytes.Buffer)
		err = jpeg.Encode(frameBuffer, img, nil)
		if err != nil {
			log.Println("[ERROR] encoding frame buffer ", err)
			continue
		}

		src := pigo.ImgToNRGBA(img)
		frame := pigo.RgbToGrayscale(src)

		cols, rows := src.Bounds().Max.X, src.Bounds().Max.Y

		cParams := pigo.CascadeParams{
			MinSize:     *minSize,
			MaxSize:     *maxSize,
			ShiftFactor: *shiftFactor,
			ScaleFactor: *scaleFactor,
			ImageParams: pigo.ImageParams{
				Pixels: frame,
				Rows:   rows,
				Cols:   cols,
				Dim:    cols,
			},
		}

		// Run the classifier over the obtained leaf nodes and return the detection results.
		// The result contains quadruplets representing the row, column, scale and detection score.
		dets := classifier.RunCascade(cParams, *angle)

		// Calculate the intersection over union (IoU) of two clusters.
		dets = classifier.ClusterDetections(dets, 0)

		dc = gg.NewContext(cols, rows)
		dc.DrawImage(src, 0, 0)

		buff := new(bytes.Buffer)
		drawMarker(dets, buff, *circleMarker)

		// Encode as MJPEG
		w.Write([]byte("Content-Type: image/jpeg\r\n"))
		w.Write([]byte("Content-Length: " + string(len(data)) + "\r\n\r\n"))
		w.Write(buff.Bytes())
		w.Write([]byte("\r\n"))
		w.Write([]byte("--informs\r\n"))
	}
	cmd.Wait()
}

// drawMarker mark the detected face region with the provided
// marker (rectangle or circle) and write it to io.Writer.
func drawMarker(detections []pigo.Detection, w io.Writer, isCircle bool) error {
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
	return dc.EncodePNG(w)
}
