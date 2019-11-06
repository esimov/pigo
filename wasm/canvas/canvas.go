package canvas

import (
	"syscall/js"
)

// Canvas struct holds the Javascript objects needed for the Canvas creation
type Canvas struct {
	done chan struct{}

	// DOM elements
	window     js.Value
	doc        js.Value
	body       js.Value
	windowSize struct{ width, height float64 }

	// Canvas properties
	canvas   js.Value
	ctx      js.Value
	reqID    js.Value
	renderer js.Func

	// Webcam properties
	navigator js.Value
	video     js.Value
}

// NewCanvas creates and initializes the new Canvas element
func NewCanvas() *Canvas {
	var c Canvas
	c.window = js.Global()
	c.doc = c.window.Get("document")
	c.body = c.doc.Get("body")

	c.windowSize.width = c.window.Get("innerWidth").Float()
	c.windowSize.height = c.window.Get("innerHeight").Float()

	c.canvas = c.doc.Call("createElement", "canvas")
	c.canvas.Set("width", c.windowSize.width)
	c.canvas.Set("height", c.windowSize.height)
	c.body.Call("appendChild", c.canvas)

	c.ctx = c.canvas.Call("getContext", "2d")

	return &c
}

// Render method calls the `requestAnimationFrame` Javascript function in asynchronous mode.
func (c *Canvas) Render() {
	go func() {
		c.renderer = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			c.reqID = c.window.Call("requestAnimationFrame", c.renderer)
			c.Log("Running")
			return nil
		})
		c.window.Call("requestAnimationFrame", c.renderer)
	}()
	<-c.done
}

// Stop stops the rendering.
func (c *Canvas) Stop() {
	c.window.Call("cancelAnimationFrame", c.reqID)
	c.done <- struct{}{}
	close(c.done)
}

func (c *Canvas) StartWebcam() *Canvas {
	c.video = c.doc.Call("createElement", "video")

	// If we don't do this, the stream will not be played.
	c.video.Set("autoplay", 1)
	c.video.Set("playsinline", 1) // important for iPhones

	// The video should fill out all of the canvas
	c.video.Set("width", 1)
	c.video.Set("height", 1)

	c.body.Call("appendChild", c.video)

	userMediaSettings := &map[string]interface{}{
		"video": true,
		"adio":  false,
	}
	stream := c.window.Get("navigator").Call("mediaDevices").Call("getUserMedia", userMediaSettings)

	c.video.Set("srcObject", stream)
	c.Render()

	return c
}

// Log calls the `console.log` Javascript
func (c *Canvas) Log(args ...interface{}) {
	c.window.Get("console").Call("log", args...)
}
