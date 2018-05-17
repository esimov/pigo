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

	sampleImg, err := pigo.GetImage("sample.png")
	if err != nil {
		log.Fatalf("Cannot open the image file: %v", err)
	}

	cParams := pigo.CascadeParams{
		MinSize: 100,
		MaxSize: 1000,
		ShiftFactor: 0.1,
		ScaleFactor: 1.1,
	}

	imgParams := pigo.ImageParams{sampleImg, 480, 640, 640}

	pigo := pigo.NewPigo()
	classifier := pigo.Unpack(cascadeFile)
	dets := classifier.RunCascade(imgParams, cParams)
	fmt.Println(dets)

}
