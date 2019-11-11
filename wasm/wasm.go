// +build js,wasm
package main

import (
	"fmt"

	"github.com/esimov/pigo/wasm/canvas"
	"github.com/esimov/pigo/wasm/detector"
)

func main() {
	c := canvas.NewCanvas()
	webcam, err := c.StartWebcam()
	if err != nil {
		c.Log(err)
	}
	det := detector.NewDetector()
	res, err := det.FetchCascade("https://raw.githubusercontent.com/esimov/pigo/master/cascade/facefinder")
	if err != nil {
		det.Log(err)
	}
	fmt.Println(res)
	webcam.Render()
}
