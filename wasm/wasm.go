// +build js,wasm
package main

import (
	"github.com/esimov/pigo/wasm/canvas"
)

func main() {
	c := canvas.NewCanvas()
	c.StartWebcam().Render()
}
