package main

import (
	"log"
	"io/ioutil"
	"github.com/esimov/pigo/pigo"
	"fmt"
)

func main() {
	cascadeFile, err := ioutil.ReadFile("data/facefinder")
	if err != nil {
		log.Fatalf("Error reading the cascade file: %s", err)
	}

	src, err := pigo.GetImage("sample.png")
	if err != nil {
		log.Fatalf("Cannot open the image file: %v", err)
	}

	sampleImg := pigo.RgbToGrayscale(src)
	fmt.Println(len(sampleImg))
	cParams := pigo.CascadeParams{
		MinSize: 20,
		MaxSize: 1000,
		ShiftFactor: 0.1,
		ScaleFactor: 1.13,
	}

	imgParams := pigo.ImageParams{sampleImg, 360, 480, 480}

	pigo := pigo.NewPigo()
	classifier := pigo.Unpack(cascadeFile)
	dets := classifier.RunCascade(imgParams, cParams)
	fmt.Println("DETECTIONS: ", len(dets))

}
