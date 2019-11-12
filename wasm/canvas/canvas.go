package canvas

import (
	"fmt"
	"math"
	"syscall/js"

	"github.com/esimov/pigo/wasm/detector"
)

// Canvas struct holds the Javascript objects needed for the Canvas creation
type Canvas struct {
	done   chan struct{}
	succCh chan struct{}
	errCh  chan error

	// DOM elements
	window     js.Value
	doc        js.Value
	body       js.Value
	windowSize struct{ width, height int }

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

	c.windowSize.width = c.window.Get("innerWidth").Int()
	c.windowSize.height = c.window.Get("innerHeight").Int()

	c.canvas = c.doc.Call("createElement", "canvas")
	c.canvas.Set("width", c.windowSize.width)
	c.canvas.Set("height", c.windowSize.height)
	c.body.Call("appendChild", c.canvas)

	c.ctx = c.canvas.Call("getContext", "2d")

	return &c
}

// Render calls the `requestAnimationFrame` Javascript function in asynchronous mode.
func (c *Canvas) Render() {
	var data = make([]byte, c.windowSize.width*c.windowSize.height*4)
	c.done = make(chan struct{})

	det := detector.NewDetector()
	if err := det.UnpackCascades(); err == nil {
		c.renderer = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			go func() {
				width, height := c.windowSize.width, c.windowSize.height
				c.reqID = c.window.Call("requestAnimationFrame", c.renderer)
				// Draw the webcam frame to the canvas element
				c.ctx.Call("drawImage", c.video, 0, 0)
				rgba := c.ctx.Call("getImageData", 0, 0, width, height).Get("data")
				//c.Log(rgba.Get("length").Int())

				uint8Arr := js.Global().Get("Uint8Array").New(rgba)
				js.CopyBytesToGo(data, uint8Arr)
				pixels := c.rgbaToGrayscale(data)
				res := det.DetectFaces(pixels, height, width)
				fmt.Println(res)
			}()
			return nil
		})
		c.window.Call("requestAnimationFrame", c.renderer)
		<-c.done
	}
}

// Stop stops the rendering.
func (c *Canvas) Stop() {
	c.window.Call("cancelAnimationFrame", c.reqID)
	c.done <- struct{}{}
	close(c.done)
}

// StartWebcam reads the webcam data and feeds it into the canvas element.
// It returns an empty struct in case of success and error in case of failure.
func (c *Canvas) StartWebcam() (*Canvas, error) {
	var err error
	c.succCh = make(chan struct{})
	c.errCh = make(chan error)

	c.video = c.doc.Call("createElement", "video")

	// If we don't do this, the stream will not be played.
	c.video.Set("autoplay", 1)
	c.video.Set("playsinline", 1) // important for iPhones

	// The video should fill out all of the canvas
	c.video.Set("width", 0)
	c.video.Set("height", 0)

	c.body.Call("appendChild", c.video)

	success := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			c.video.Set("srcObject", args[0])
			c.video.Call("play")
			c.succCh <- struct{}{}
		}()
		return nil
	})

	failure := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			err = fmt.Errorf("failed initialising the camera: %s", args[0].String())
			c.errCh <- err
		}()
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

	select {
	case <-c.succCh:
		return c, nil
	case err := <-c.errCh:
		return nil, err
	}
}

func (c *Canvas) rgbaToGrayscale(data []uint8) []uint8 {
	rows, cols := c.windowSize.width, c.windowSize.height

	// for i := 0; i < rows*cols; i++ {
	// 	r := data[i*4+0]
	// 	g := data[i*4+1]
	// 	b := data[i*4+2]
	// 	var gray = uint8(math.Round(0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)))
	// 	data[i*4+0] = gray
	// 	data[i*4+1] = gray
	// 	data[i*4+2] = gray
	// }

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			// gray = 0.2*red + 0.7*green + 0.1*blue
			data[r*cols+c] = uint8(math.Round(
				0.2126*float64(data[r*4*cols+4*c+0]) +
					0.7152*float64(data[r*4*cols+4*c+1]) +
					0.0722*float64(data[r*4*cols+4*c+2])))
		}
	}
	return data
}

// Log calls the `console.log` Javascript function
func (c *Canvas) Log(args ...interface{}) {
	c.window.Get("console").Call("log", args...)
}

// Alert calls the `alert` Javascript function
func (c *Canvas) Alert(args ...interface{}) {
	alert := c.window.Get("alert")
	alert.Invoke(args...)
}
