package main

import (
	"fmt"
	"image"
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
	demo "github.com/esimov/pigo/demos/native"
	pigo "github.com/esimov/pigo/pkg"
	"github.com/fogleman/gg"
	"github.com/pion/mediadevices"
	_ "github.com/pion/mediadevices/pkg/driver/camera" // This is required to register camera adapter
	"github.com/pion/mediadevices/pkg/io/video"
	"github.com/pion/mediadevices/pkg/prop"
)

const (
	windowWidth  = 1024
	windowHeight = 640
)

var (
	c     *demo.Constructor
	media video.Reader
)

func main() {
	classifier := pigo.UnpackCascade()

	mediaStream, err := mediadevices.GetUserMedia(mediadevices.MediaStreamConstraints{
		Video: func(c *mediadevices.MediaTrackConstraints) {
			c.Width = prop.Int(windowWidth)
			c.Height = prop.Int(windowHeight)
		},
	})
	if err != nil {
		log.Fatalf("failed to detect user media: %v", err)
	}

	imgCtx := gg.NewContext(windowWidth, windowHeight)
	c, err = demo.NewConstructor(imgCtx, classifier, nil, nil)
	if err != nil {
		log.Fatalf("failed to instantiate constructor: %v", err)
	}

	// since we're trying to access the raw data, we need to cast Track to its real type, *mediadevices.VideoTrack
	videoTrack := mediaStream.GetVideoTracks()[0].(*mediadevices.VideoTrack)
	defer videoTrack.Close()

	media = videoTrack.NewReader(false)

	go func() {
		w := new(app.Window)
		w.Option(
			app.Title("PIGO - Face detection demo"),
			app.Size(unit.Dp(windowWidth), unit.Dp(windowHeight)),
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

			frame, release, err := media.Read()
			if err != nil {
				return fmt.Errorf("error encountered reading video frame: %w", err)
			}

			if img, ok := frame.(*image.YCbCr); ok {
				scaleX := float64(windowWidth) / float64(img.Bounds().Dx())
				scaleY := float64(windowHeight) / float64(img.Bounds().Dy())

				c.ImgCtx.Push()
				c.ImgCtx.Scale(scaleX, scaleY)
				c.ImgCtx.DrawImage(frame, 0, 0)

				c.DetectAndDraw(img)

				c.ImgCtx.Pop()
			}
			release()

			src := paint.NewImageOp(c.ImgCtx.Image())
			src.Add(gtx.Ops)

			layout.Stack{}.Layout(gtx,
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					imageWidget := widget.Image{
						Src:      src,
						Fit:      widget.Fill,
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
