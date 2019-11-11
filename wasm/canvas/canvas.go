package canvas

import (
	"fmt"
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
	var data = make([]byte, int(c.windowSize.width*c.windowSize.height*4))

	go func() {
		c.renderer = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			c.reqID = c.window.Call("requestAnimationFrame", c.renderer)
			// Draw the canvas frame to the canvas element
			c.ctx.Call("drawImage", c.video, 0, 0)
			rgba := c.ctx.Call("getImageData", 0, 0, c.windowSize.width, c.windowSize.height).Get("data")
			c.Log(rgba.Get("length").Int())

			uint8Arr := js.Global().Get("Uint8Array").New(rgba)
			js.CopyBytesToGo(data, uint8Arr)

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

func (c *Canvas) StartWebcam() (*Canvas, error) {
	var err error
	c.video = c.doc.Call("createElement", "video")

	// If we don't do this, the stream will not be played.
	c.video.Set("autoplay", 1)
	c.video.Set("playsinline", 1) // important for iPhones

	// The video should fill out all of the canvas
	c.video.Set("width", 0)
	c.video.Set("height", 0)

	c.body.Call("appendChild", c.video)

	success := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		c.video.Set("srcObject", args[0])
		c.video.Call("play")
		return nil
	})

	failure := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		err = fmt.Errorf("failed initialising the camera: %s", args[0].String())
		return nil
	})

	opts := js.Global().Get("Object").New()
	widthOpts := js.Global().Get("Object").New()
	widthOpts.Set("min", 1024)
	widthOpts.Set("max", 1920)

	heightOpts := js.Global().Get("Object").New()
	heightOpts.Set("min", 720)
	heightOpts.Set("max", 1080)

	videoSize := js.Global().Get("Object").New()
	videoSize.Set("width", widthOpts)
	videoSize.Set("height", heightOpts)
	videoSize.Set("aspectRatio", 1.777777778)

	opts.Set("video", videoSize)
	opts.Set("audio", false)

	promise := c.window.Get("navigator").Get("mediaDevices").Call("getUserMedia", opts)
	promise.Call("then", success, failure)

	return c, err
}

// Log calls the `console.log` Javascript
func (c *Canvas) Log(args ...interface{}) {
	c.window.Get("console").Call("log", args...)
}
