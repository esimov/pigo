package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	pigo "github.com/esimov/pigo/pkg"
	"github.com/pion/mediadevices"
	_ "github.com/pion/mediadevices/pkg/driver/camera" // This is required to register camera adapter
	"github.com/pion/mediadevices/pkg/io/video"
	"github.com/pion/mediadevices/pkg/prop"
)

const (
	windowWidth  = 1024
	windowHeight = 640

	confidenceLevel = 5.0
)

var (
	classifier  *pigo.Pigo
	videoReader video.Reader
)

func main() {
	p := pigo.NewPigo()
	classifier = p.Unpack()

	mediaStream, err := mediadevices.GetUserMedia(mediadevices.MediaStreamConstraints{
		Video: func(c *mediadevices.MediaTrackConstraints) {
			c.Width = prop.Int(windowWidth)
			c.Height = prop.Int(windowHeight)
		},
	})
	if err != nil {
		panic(err)
	}

	// since we're trying to access the raw data, we need to cast Track to its real type, *mediadevices.VideoTrack
	videoTrack := mediaStream.GetVideoTracks()[0].(*mediadevices.VideoTrack)
	defer videoTrack.Close()

	videoReader = videoTrack.NewReader(false)

	go func() {
		w := new(app.Window)
		w.Option(
			app.Title("Face detection demo"),
			app.Size(unit.Dp(windowWidth), unit.Dp(windowHeight)),
			app.MinSize(unit.Dp(windowWidth), unit.Dp(windowHeight)),
			app.MaxSize(unit.Dp(windowWidth), unit.Dp(windowHeight)),
		)

		// Center window to the middle of the screen
		w.Perform(system.ActionCenter)

		if err := run(w); err != nil {
			log.Fatal(err)
		}

		os.Exit(0)
	}()

	app.Main()
}

func run(w *app.Window) error {
	var ops op.Ops

	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			// This graphics context is used for managing the rendering state.
			gtx := app.NewContext(&ops, e)

			frame, release, err := videoReader.Read()
			if err != nil {
				return fmt.Errorf("error encountered reading the frame: %w", err)
			}

			if img, ok := frame.(*image.YCbCr); ok {
				dets := detectFace(img)
				for _, det := range dets {
					if det.Q >= confidenceLevel {
						drawRectangle(img,
							image.Rect(det.Col-det.Scale/2, det.Row-det.Scale/2, det.Col+det.Scale/2, det.Row+det.Scale/2),
							color.RGBA{R: 0xff, A: 0xff})

					}
				}
			}

			release()

			src := paint.NewImageOp(frame)
			src.Add(gtx.Ops)

			layout.Stack{}.Layout(gtx,
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					imageWidget := widget.Image{
						Src:      src,
						Scale:    1 / float32(unit.Dp(1)),
						Fit:      widget.Contain,
						Position: layout.Center,
					}

					return imageWidget.Layout(gtx)
				}),
			)

			for {
				event, ok := gtx.Event(key.Filter{
					Name: key.NameEscape,
				})
				if !ok {
					break
				}
				switch event := event.(type) {
				case key.Event:
					switch event.Name {
					case key.NameEscape:
						w.Perform(system.ActionClose)
						return nil
					}
				}
			}

			// Pass the drawing operations to the GPU.
			e.Frame(gtx.Ops)
			// Redraw the window immediately.
			gtx.Execute(op.InvalidateCmd{})
		}
	}
}

func detectFace(frame *image.YCbCr) []pigo.Detection {
	bounds := frame.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	cascadeParams := pigo.CascadeParams{
		MinSize:     100,
		MaxSize:     min(width, height),
		ShiftFactor: 0.15,
		ScaleFactor: 1.1,
		ImageParams: pigo.ImageParams{
			Pixels: frame.Y, // Y in YCbCr should be enough to detect faces
			Rows:   height,
			Cols:   width,
			Dim:    width,
		},
	}

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	dets := classifier.RunCascade(cascadeParams, 0.0)

	// Calculate the intersection over union (IoU) of two clusters.
	dets = classifier.ClusterDetections(dets, 0)

	return dets
}

func drawRectangle(img *image.YCbCr, rect image.Rectangle, c color.Color) {
	for x := rect.Min.X; x < rect.Max.X; x++ {
		setIfInside(img, x, rect.Min.Y, c)
		setIfInside(img, x, rect.Max.Y-1, c)
	}

	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		setIfInside(img, rect.Min.X, y, c)
		setIfInside(img, rect.Max.X-1, y, c)
	}
}

func setIfInside(img *image.YCbCr, x int, y int, c color.Color) {
	point := image.Pt(x, y)

	if !point.In(img.Bounds()) {
		return
	}

	r, g, b, _ := c.RGBA()
	yy, cb, cr := color.RGBToYCbCr(uint8(r>>8), uint8(g>>8), uint8(b>>8))

	img.Y[img.YOffset(x, y)] = yy

	offset := img.COffset(x, y)
	img.Cb[offset] = cb
	img.Cr[offset] = cr
}
