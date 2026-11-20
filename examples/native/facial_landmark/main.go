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
				return fmt.Errorf("error encountered reading video frame: %w", err)
			}

			if img, ok := frame.(*image.YCbCr); ok {
				imgCtx.DrawImage(frame, 0, 0)

				dets := detectFace(img)
				for _, det := range dets {
					if det.Q >= confidenceLevel {
						drawRectangle(img,
							image.Rect(det.Col-det.Scale/2, det.Row-det.Scale/2, det.Col+det.Scale/2, det.Row+det.Scale/2),
							color.RGBA{R: 0xff, A: 0xff})

						leftPupil := detectLeftPupil(det)
						rect := image.Rect(
							det.Col-det.Scale/2,
							det.Row-det.Scale/2,
							det.Col+det.Scale/2,
							det.Row+det.Scale/2,
						)
						rows, cols := rect.Max.X-rect.Min.X, rect.Max.Y-rect.Min.Y
						if leftPupil.Row > 0 && leftPupil.Col > 0 {
							fmt.Println("left pupil detected")
							drawEyeDetectionMarker(imgCtx,
								float64(cols/2-(det.Col-leftPupil.Col)),
								float64(rows/2-(det.Row-leftPupil.Row)),
								float64(leftPupil.Scale),
								color.RGBA{R: 255, G: 0, B: 0, A: 255},
								true,
							)
						}

						rightPupil := detectRightPupil(det)
						if rightPupil.Row > 0 && rightPupil.Col > 0 {
							fmt.Println("right pupil detected")
						}
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
						Fit:      widget.Cover,
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

// drawEyeDetectionMarker is a helper function to draw the detection marks
func drawEyeDetectionMarker(ctx *gg.Context, x, y, r float64, c color.RGBA, markDet bool) {
	ctx.DrawArc(x, y, r*0.15, 0, 2*math.Pi)
	ctx.SetFillStyle(gg.NewSolidPattern(c))
	ctx.Fill()

	if markDet {
		ctx.DrawRectangle(x-(r*1.5), y-(r*1.5), r*3, r*3)
		ctx.SetLineWidth(2.0)
		ctx.SetStrokeStyle(gg.NewSolidPattern(color.RGBA{R: 255, G: 255, B: 0, A: 255}))
		ctx.Stroke()
	}
}
