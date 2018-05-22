package main

import (
	"log"
	"io/ioutil"
	"github.com/esimov/pigo/pigo"
	"image/color"
	"github.com/fogleman/gg"
)
var dc *gg.Context

func main() {
	cascadeFile, err := ioutil.ReadFile("data/facefinder")
	if err != nil {
		log.Fatalf("Error reading the cascade file: %s", err)
	}

	src, err := pigo.GetImage("me.jpg")
	if err != nil {
		log.Fatalf("Cannot open the image file: %v", err)
	}

	sampleImg := pigo.RgbToGrayscale(src)

	cParams := pigo.CascadeParams{
		MinSize: 20,
		MaxSize: 1000,
		ShiftFactor: 0.1,
		ScaleFactor: 1.13,
	}
	cols, rows := src.Bounds().Max.X, src.Bounds().Max.Y
	imgParams := pigo.ImageParams{sampleImg, rows, cols, cols}

	pigo := pigo.NewPigo()
	// Unpack the binary file. This will return the number of cascade trees,
	// the tree depth, the threshold and the prediction from tree's leaf nodes.
	classifier := pigo.Unpack(cascadeFile)

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	dets := classifier.RunCascade(imgParams, cParams)

	// Calculate the intersection over union (IoU) for two
	dets = classifier.ClusterDetections(classifier.RunCascade(imgParams, cParams), 0.2)

	dc = gg.NewContext(cols, rows)
	dc.DrawImage(src, 0, 0)

	if err := output(dets); err != nil {
		log.Fatalf("Cannot save the output image %v", err)
	}
}

func output(detections []pigo.Detection) error {
	var qThresh float32 = 5.0

	for i := 0; i < len(detections); i++ {
		if detections[i].Q > qThresh {
			dc.DrawRectangle(
				float64(detections[i].Col-detections[i].Scale /2),
				float64(detections[i].Row-detections[i].Scale /2),
				float64(detections[i].Scale),
				float64(detections[i].Scale),
			)
			dc.SetLineWidth(3.0)
			dc.SetStrokeStyle(gg.NewSolidPattern(color.RGBA{R: 255, G: 0, B: 0, A: 255}))
			dc.Stroke()
		}
	}

	return dc.SavePNG("out.png")
}
