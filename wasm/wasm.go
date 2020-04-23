// +build js,wasm

package main

import (
	"fmt"

	"github.com/esimov/pigo/wasm/canvas"
)

func main() {
	c := canvas.NewCanvas()
	webcam, err := c.StartWebcam()
	if err != nil {
		c.Alert("Webcam not detected!")
	} else {
		err := webcam.Render()
		if err != nil {
			c.Alert(fmt.Sprint(err))
		}
	}
}
