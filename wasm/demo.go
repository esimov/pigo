// +build js,wasm

package main

import "github.com/esimov/pigo/wasm/demo"

func main() {
	c := demo.NewCanvas()
	webcam, err := c.StartWebcam()
	if err != nil {
		c.Alert("Webcam not detected!")
	} else {
		webcam.Render()
	}
}
