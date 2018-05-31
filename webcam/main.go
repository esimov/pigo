package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"os/exec"

	"github.com/esimov/pigo/pigo"
	"github.com/fogleman/gg"
)

const boundary = "informs"

var dc *gg.Context

func main() {
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

	cascadeFile, err := ioutil.ReadFile("../data/facefinder")
	if err != nil {
		log.Fatalf("Error reading the cascade file: %v", err)
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

		data, err := ioutil.ReadAll(p)
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

		cParams := pigo.CascadeParams{
			MinSize:     100,
			MaxSize:     1000,
			ShiftFactor: 0.1,
			ScaleFactor: 1.1,
		}
		cols, rows := src.Bounds().Max.X, src.Bounds().Max.Y
		imgParams := pigo.ImageParams{frame, rows, cols, cols}

		pigo := pigo.NewPigo()
		// Unpack the binary file. This will return the number of cascade trees,
		// the tree depth, the threshold and the prediction from tree's leaf nodes.
		classifier := pigo.Unpack(cascadeFile)

		// Run the classifier over the obtained leaf nodes and return the detection results.
		// The result contains quadruplets representing the row, column, scale and detection score.
		dets := classifier.RunCascade(imgParams, cParams)

		// Calculate the intersection over union (IoU) of two clusters.
		dets = classifier.ClusterDetections(dets, 0)
		fmt.Println(dets)

		dc = gg.NewContext(cols, rows)
		dc.DrawImage(src, 0, 0)

		buff := new(bytes.Buffer)
		if err := drawMarker(dets, buff, false); err != nil {
			log.Println("Cannot save the output image %v", err)
		}
		buff.Write(frameBuffer.Bytes())
		// just MJPEG
		w.Write([]byte("Content-Type: image/jpeg\r\n"))
		w.Write([]byte("Content-Length: " + string(len(data)) + "\r\n\r\n"))
		w.Write(buff.Bytes())
		w.Write([]byte("\r\n"))
		w.Write([]byte("--informs\r\n"))
	}
	cmd.Wait()
}

// drawMarker mark the face region with the provided marker (rectangle or circle) and write it to io.Writer.
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
