package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
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
	"github.com/fogleman/gg"
	"github.com/pion/mediadevices"
	_ "github.com/pion/mediadevices/pkg/driver/camera" // This is required to register camera adapter
	"github.com/pion/mediadevices/pkg/io/video"
	"github.com/pion/mediadevices/pkg/prop"
)

const (
	windowWidth  = 1024
	windowHeight = 640

	confidenceLevel = 5.0
	perturbation    = 63
)

var (
	videoReader    video.Reader
	faceClassifier *pigo.Pigo
	puplocCascade  *pigo.PuplocCascade
	imgParams      *pigo.ImageParams
	imgCtx         *gg.Context

	showPupilDet bool = true
)

func main() {
	var err error

	faceClassifier = pigo.UnpackCascade()
	puplocCascade, err = pigo.UnpackPlc()
	if err != nil {
		panic(err)
	}

	mediaStream, err := mediadevices.GetUserMedia(mediadevices.MediaStreamConstraints{
		Video: func(c *mediadevices.MediaTrackConstraints) {
			c.Width = prop.Int(windowWidth)
			c.Height = prop.Int(windowHeight)
		},
	})
	if err != nil {
		panic(err)
	}

	imgCtx = gg.NewContext(windowWidth, windowHeight)

	// since we're trying to access the raw data, we need to cast Track to its real type, *mediadevices.VideoTrack
	videoTrack := mediaStream.GetVideoTracks()[0].(*mediadevices.VideoTrack)
	defer videoTrack.Close()

	videoReader = videoTrack.NewReader(false)

	go func() {
		w := new(app.Window)
		w.Option(
			app.Title("PIGO - Puploc demo"),
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

			frame, release, err := videoReader.Read()
			if err != nil {
				return fmt.Errorf("error encountered reading video frame: %w", err)
			}

			if img, ok := frame.(*image.YCbCr); ok {
				scaleX := float64(windowWidth) / float64(img.Bounds().Dx())
				scaleY := float64(windowHeight) / float64(img.Bounds().Dy())

				imgCtx.Push()
				imgCtx.Scale(scaleX, scaleY)
				imgCtx.DrawImage(frame, 0, 0)

				dets := detectFaces(img)
				for _, det := range dets {
					if det.Q >= confidenceLevel {
						drawFaceRect(imgCtx, float64(det.Col-det.Scale/2),
							float64(det.Row-det.Scale/2),
							float64(det.Scale),
							float64(det.Scale),
							color.RGBA{R: 255, G: 0, B: 0, A: 255})

						if !showPupilDet {
							continue
						}

						leftPupil := detectLeftPupil(det)
						if leftPupil.Row > 0 && leftPupil.Col > 0 {
							drawPupilDet(imgCtx,
								float64(leftPupil.Col),
								float64(leftPupil.Row),
								float64(leftPupil.Scale),
								color.RGBA{R: 255, G: 0, B: 0, A: 255},
							)
						}

						rightPupil := detectRightPupil(det)
						if rightPupil.Row > 0 && rightPupil.Col > 0 {
							drawPupilDet(imgCtx,
								float64(rightPupil.Col),
								float64(rightPupil.Row),
								float64(rightPupil.Scale),
								color.RGBA{R: 255, G: 0, B: 0, A: 255},
							)
						}
					}
				}
				imgCtx.Pop()
			}
			release()

			src := paint.NewImageOp(imgCtx.Image())
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
				event, ok := gtx.Event(
					key.Filter{Name: key.NameEscape},
					key.Filter{Name: key.Name("S")},
				)
				if !ok {
					break
				}

				switch event := event.(type) {
				case key.Event:
					if event.State == key.Press {
						switch event.Name {
						case key.NameEscape:
							w.Perform(system.ActionClose)
							return nil
						case key.Name("S"):
							showPupilDet = !showPupilDet
						}
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

func detectFaces(frame *image.YCbCr) []pigo.Detection {
	bounds := frame.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	imgParams = &pigo.ImageParams{
		Pixels: frame.Y, // Y in YCbCr should be enough to detect faces
		Rows:   height,
		Cols:   width,
		Dim:    width,
	}

	cascadeParams := pigo.CascadeParams{
		MinSize:     100,
		MaxSize:     min(width, height),
		ShiftFactor: 0.15,
		ScaleFactor: 1.1,
		ImageParams: *imgParams,
	}

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	dets := faceClassifier.RunClassifier(cascadeParams, 0.0)

	// Calculate the intersection over union (IoU) of two clusters.
	dets = faceClassifier.ClusterDetections(dets, 0.1)

	return dets
}

func detectLeftPupil(face pigo.Detection) *pigo.Puploc {
	puploc := &pigo.Puploc{
		Row:      face.Row - int(0.075*float32(face.Scale)),
		Col:      face.Col - int(0.175*float32(face.Scale)),
		Scale:    float32(face.Scale) * 0.25,
		Perturbs: perturbation,
	}

	det := puplocCascade.RunClassifier(puploc, *imgParams, 0.0, false)
	if det.Row > 0 && det.Col > 0 {
		return det
	}

	return nil
}

func detectRightPupil(face pigo.Detection) *pigo.Puploc {
	puploc := &pigo.Puploc{
		Row:      face.Row - int(0.075*float32(face.Scale)),
		Col:      face.Col + int(0.185*float32(face.Scale)),
		Scale:    float32(face.Scale) * 0.25,
		Perturbs: perturbation,
	}

	rightPupil := puplocCascade.RunClassifier(puploc, *imgParams, 0.0, false)
	if rightPupil.Row > 0 && rightPupil.Col > 0 {
		return rightPupil
	}

	return nil
}

func drawFaceRect(ctx *gg.Context, x, y, w, h float64, c color.RGBA) {
	ctx.DrawRectangle(x, y, w, h)
	ctx.SetLineWidth(2.0)
	ctx.SetStrokeStyle(gg.NewSolidPattern(c))
	ctx.Stroke()
}

func drawPupilDet(ctx *gg.Context, x, y, w float64, c color.RGBA) {
	ctx.DrawArc(x, y, w*0.15, 0, 2*math.Pi)
	ctx.SetFillStyle(gg.NewSolidPattern(c))
	ctx.Fill()
}
