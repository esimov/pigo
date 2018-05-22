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
	
	cParams := pigo.CascadeParams{
		MinSize: 20,
		MaxSize: 1000,
		ShiftFactor: 0.1,
		ScaleFactor: 1.13,
	}
	cols, rows := src.Bounds().Max.X, src.Bounds().Max.Y
	imgParams := pigo.ImageParams{sampleImg, rows, cols, cols}

	pigo := pigo.NewPigo()
	classifier := pigo.Unpack(cascadeFile)
	dets := classifier.RunCascade(imgParams, cParams)
	fmt.Println(len(dets))

}
