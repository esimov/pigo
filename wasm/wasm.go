// +build js,wasm
package main

import (
	"fmt"

	"github.com/esimov/pigo/wasm/canvas"
	"github.com/esimov/pigo/wasm/detector"
)

func main() {
	c := canvas.NewCanvas()
	if webcam, err := c.StartWebcam(); err != nil {
		webcam.Render()
	}

	det := detector.NewDetector()
	res, err := det.FetchCascade("https://raw.githubusercontent.com/esimov/pigo/master/cascade/facefinder")
	if err != nil {
		det.Log(err)
	}
	fmt.Println(res)
}
