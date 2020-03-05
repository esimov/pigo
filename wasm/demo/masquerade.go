package demo

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"syscall/js"
	"time"

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

	showPupil  bool
	showFace   bool
	showMask   bool
	drawCircle bool
}

type point struct {
	x, y int
}

var (
	images = make([]js.Value, 6)
	files  = []string{
		"/demo/images/neon-yellow.png",
		"/demo/images/sunglasses.png",
		"/demo/images/neon-green.png",
		"/demo/images/carnival.png",
		"/demo/images/carnival2.png",
		"/demo/images/neon-disco.png",
	}
	curImgWidth  int
	curImgHeight int
	imgIdx       int
)

var det *detector.Detector

// NewCanvas creates and initializes the new Canvas element
func NewCanvas() *Canvas {
	var c Canvas
	c.window = js.Global()
	c.doc = c.window.Get("document")
	c.body = c.doc.Get("body")

	c.windowSize.width = 1280
	c.windowSize.height = 720

	c.canvas = c.doc.Call("createElement", "canvas")
	c.canvas.Set("width", js.ValueOf(c.windowSize.width))
	c.canvas.Set("height", js.ValueOf(c.windowSize.height))
	c.canvas.Set("id", "canvas")
	c.body.Call("appendChild", c.canvas)

	c.ctx = c.canvas.Call("getContext", "2d")
	c.showPupil = true
	c.showFace = true
	c.showMask = true
	c.drawCircle = false

	det = detector.NewDetector()
	return &c
}

// Render calls the `requestAnimationFrame` Javascript function in asynchronous mode.
func (c *Canvas) Render() {
	var data = make([]byte, c.windowSize.width*c.windowSize.height*4)
	c.done = make(chan struct{})

	for i, file := range files {
		img := c.loadImage(file)
		images[i] = js.Global().Call("eval", "new Image()")
		images[i].Set("src", "data:image/png;base64,"+img)
	}

	curImgWidth = js.ValueOf(images[0].Get("naturalWidth")).Int()
	curImgHeight = js.ValueOf(images[0].Get("naturalHeight")).Int()

	if err := det.UnpackCascades(); err == nil {
		c.renderer = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			go func() {
				c.window.Get("stats").Call("begin")

				width, height := c.windowSize.width, c.windowSize.height
				c.reqID = c.window.Call("requestAnimationFrame", c.renderer)
				// Draw the webcam frame to the canvas element
				c.ctx.Call("drawImage", c.video, 0, 0)
				rgba := c.ctx.Call("getImageData", 0, 0, width, height).Get("data")

				// Convert the rgba value of type Uint8ClampedArray to Uint8Array in order to
				// be able to transfer it from Javascript to Go via the js.CopyBytesToGo function.
				uint8Arr := js.Global().Get("Uint8Array").New(rgba)
				js.CopyBytesToGo(data, uint8Arr)
				pixels := c.rgbaToGrayscale(data)
				res := det.DetectFaces(pixels, height, width)
				c.drawDetection(res)

				c.window.Get("stats").Call("end")
			}()
			return nil
		})
		c.window.Call("requestAnimationFrame", c.renderer)
		c.detectKeyPress()
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

	videoSize := js.Global().Get("Object").New()
	videoSize.Set("width", c.windowSize.width)
	videoSize.Set("height", c.windowSize.height)
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

// rgbaToGrayscale converts the rgb pixel values to grayscale
func (c *Canvas) rgbaToGrayscale(data []uint8) []uint8 {
	rows, cols := c.windowSize.width, c.windowSize.height
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

// drawDetection draws the detected faces and eyes.
func (c *Canvas) drawDetection(dets [][]int) {
	var p1, p2 point
	var imgScale float64

	for i := 0; i < len(dets); i++ {
		if dets[i][3] > 50 {
			row, col, scale := dets[i][1], dets[i][0], dets[i][2]
			c.ctx.Call("beginPath")
			c.ctx.Set("lineWidth", 3)
			c.ctx.Set("strokeStyle", "red")

			if c.showFace {
				if c.drawCircle {
					c.ctx.Call("moveTo", row+int(scale/2), col)
					c.ctx.Call("arc", row, col, scale/2, 0, 2*math.Pi, true)
				} else {
					c.ctx.Call("rect", row-scale/2, col-scale/2, scale, scale)
				}
			}
			c.ctx.Call("stroke")

			if c.showPupil {
				leftPupil := det.DetectLeftPupil(dets[i])
				if leftPupil != nil {
					col, row, scale := leftPupil.Col, leftPupil.Row, leftPupil.Scale/8
					c.ctx.Call("moveTo", col+int(scale), row)
					c.ctx.Call("arc", col, row, scale, 0, 2*math.Pi, true)

					p1 = point{x: leftPupil.Row, y: leftPupil.Col}
				}

				rightPupil := det.DetectRightPupil(dets[i])
				if rightPupil != nil {
					col, row, scale := rightPupil.Col, rightPupil.Row, rightPupil.Scale/8
					c.ctx.Call("moveTo", col+int(scale), row)
					c.ctx.Call("arc", col, row, scale, 0, 2*math.Pi, true)

					p2 = point{x: rightPupil.Row, y: rightPupil.Col}
				}
				c.ctx.Call("stroke")

				if c.showMask && (p1.x != 0 && p2.y != 0) {
					// Calculate the lean angle between the pupils.
					angle := 1 - (math.Atan2(float64(p2.y-p1.y), float64(p2.x-p1.x)) * 180 / math.Pi / 90)
					if scale < curImgWidth || scale < curImgHeight {
						if curImgHeight > curImgWidth {
							imgScale = float64(scale) / float64(curImgHeight)
						} else {
							imgScale = float64(scale) / float64(curImgWidth)
						}
					}

					width, height := float64(curImgWidth)*imgScale, float64(curImgHeight)*imgScale
					tx := row - int(width/2)
					ty := leftPupil.Row + (leftPupil.Row-rightPupil.Row)/2 - int(height/2)

					c.ctx.Call("save")
					c.ctx.Call("translate", js.ValueOf(tx).Int(), js.ValueOf(ty).Int())
					c.ctx.Call("rotate", js.ValueOf(angle).Float())
					c.ctx.Call("drawImage", images[imgIdx],
						js.ValueOf(0).Int(), js.ValueOf(0).Int(),
						js.ValueOf(width).Int(), js.ValueOf(height).Int(),
					)
					c.ctx.Call("restore")
				}
			}
		}
	}
}

// detectKeyPress listen for the keypress event and retrieves the key code.
func (c *Canvas) detectKeyPress() {
	keyEventHandler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		keyCode := args[0].Get("key")
		switch {
		case keyCode.String() == "q":
			c.showFace = !c.showFace
		case keyCode.String() == "e":
			c.showPupil = !c.showPupil
		case keyCode.String() == "a":
			c.drawCircle = !c.drawCircle
		case keyCode.String() == "r":
			c.showMask = !c.showMask
		case keyCode.String() == "w":
			imgIdx++
			if imgIdx > len(images)-1 {
				imgIdx = 0
			}
			curImgWidth = js.ValueOf(images[imgIdx].Get("naturalWidth")).Int()
			curImgHeight = js.ValueOf(images[imgIdx].Get("naturalHeight")).Int()
		case keyCode.String() == "s":
			imgIdx--
			if imgIdx < 0 {
				imgIdx = len(images) - 1
			}
			curImgWidth = js.ValueOf(images[imgIdx].Get("naturalWidth")).Int()
			curImgHeight = js.ValueOf(images[imgIdx].Get("naturalHeight")).Int()
		default:
			c.drawCircle = false
		}
		return nil
	})
	c.doc.Call("addEventListener", "keypress", keyEventHandler)
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

// loadImage load the source image and encodes it to base64 format.
func (c *Canvas) loadImage(path string) string {
	href := js.Global().Get("location").Get("href")
	u, err := url.Parse(href.String())
	if err != nil {
		log.Fatal(err)
	}
	u.Path = path
	u.RawQuery = fmt.Sprint(time.Now().UnixNano())

	log.Println("loading image file: " + u.String())
	resp, err := http.Get(u.String())
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(b)
}
