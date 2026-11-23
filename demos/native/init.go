package demo

import (
	"fmt"
	"image"
	"image/color"
	"math"

	pigo "github.com/esimov/pigo/pkg"
	"github.com/fogleman/gg"
)

const (
	perturbation    = 75
	confidenceLevel = 4.0
)

var (
	eyeCascades   = [5]string{"lp46", "lp44", "lp42", "lp38", "lp312"}
	mouthCascades = [4]string{"lp93", "lp84", "lp82", "lp81"}
)

type Constructor struct {
	ImgCtx          *gg.Context
	classifier      *pigo.Pigo
	puploc          *pigo.PuplocCascade
	flpcs           map[string][]*pigo.FlpCascade
	imgParams       *pigo.ImageParams
	detectedFaces   []*faceTracker
	ShowPupilDet    bool
	ShowLandmarkDet bool
}

type faceTracker struct {
	x, y, scale float64
	detected    bool
	leftPupil   *pupilTracker
	rightPupil  *pupilTracker
}

type pupilTracker struct {
	x, y, scale float64
	detected    bool
	missed      int
}

type DetectionOptions func(*Constructor) error

func NewConstructor(
	imgCtx *gg.Context,
	faceClassifier *pigo.Pigo,
	puplocCascade *pigo.PuplocCascade,
	flpcs map[string][]*pigo.FlpCascade,
	opts ...DetectionOptions,
) (*Constructor, error) {
	constructor := &Constructor{
		ImgCtx:     imgCtx,
		classifier: faceClassifier,
		puploc:     puplocCascade,
		flpcs:      flpcs,
	}

	for _, opt := range opts {
		if err := opt(constructor); err != nil {
			return nil, fmt.Errorf("failed to apply the constructor options: %w", err)
		}
	}

	return constructor, nil
}

// WithPupilDetection instantiates the constructor with pupil detection enabled.
func WithPupilDetection() DetectionOptions {
	return func(c *Constructor) error {
		c.ShowPupilDet = true
		return nil
	}
}

// WithLandmarkDetection instantiates the constructor with the landmark points detection enabled.
func WithLandmarkDetection() DetectionOptions {
	return func(c *Constructor) error {
		c.ShowPupilDet = true
		c.ShowLandmarkDet = true
		return nil
	}
}

func (c *Constructor) DetectAndDraw(frame *image.YCbCr) {
	bounds := frame.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	c.imgParams = &pigo.ImageParams{
		Pixels: frame.Y, // Y in YCbCr should be enough to detect faces
		Rows:   height,
		Cols:   width,
		Dim:    width,
	}

	cascadeParams := &pigo.CascadeParams{
		MinSize:     100,
		MaxSize:     min(width, height),
		ShiftFactor: 0.15,
		ScaleFactor: 1.1,
		ImageParams: *c.imgParams,
	}

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	dets := c.classifier.RunClassifier(cascadeParams, 0.0)

	// Calculate the intersection over union (IoU) of two clusters.
	dets = c.classifier.ClusterDetections(dets, 0.1)

	// detectedFaces assumes detection i in the current frame is
	// the same face as detection i in the previous frame.
	// With multiple detected faces, that assumption breaks easily,
	// resulting in rectangles smoothed toward the wrong previous position.
	used := make([]bool, len(c.detectedFaces))
	for _, det := range dets {
		if det.Q < confidenceLevel {
			continue
		}

		best := math.MaxFloat64
		var face *faceTracker
		for i, f := range c.detectedFaces {
			if used[i] {
				continue
			}

			dx := f.x - float64(det.Col)
			dy := f.y - float64(det.Row)
			dist := math.Sqrt(dx*dx + dy*dy)

			if dist < best {
				best = dist
				face = f
			}
		}

		if math.Sqrt(best) > float64(det.Scale) {
			face = &faceTracker{}

			c.detectedFaces = append(c.detectedFaces, face)
			used = append(used, true)
		} else {
			for i, f := range c.detectedFaces {
				if f == face {
					used[i] = true
					break
				}
			}
		}

		stabilizeFace(face, det)
		c.DrawRect(face.x-face.scale/2, face.y-face.scale/2,
			face.scale, face.scale,
			color.RGBA{R: 255, G: 0, B: 0, A: 255})

		if !c.ShowPupilDet {
			continue
		}

		leftPupil := c.detectLeftPupil(det)
		if leftPupil != nil {
			c.DrawArc(float64(leftPupil.Col),
				float64(leftPupil.Row),
				float64(leftPupil.Scale*1.5),
				color.RGBA{R: 255, G: 0, B: 0, A: 255},
			)
		}

		rightPupil := c.detectRightPupil(det)
		if rightPupil != nil {
			c.DrawArc(float64(rightPupil.Col),
				float64(rightPupil.Row),
				float64(rightPupil.Scale*1.5),
				color.RGBA{R: 255, G: 0, B: 0, A: 255},
			)
		}

		if !c.ShowLandmarkDet {
			continue
		}

		for _, eye := range eyeCascades {
			for _, flpc := range c.flpcs[eye] {
				flp := flpc.LandmarkPoint(leftPupil, rightPupil, c.imgParams, perturbation, false)
				if flp.Row > 0 && flp.Col > 0 {
					c.DrawArc(float64(flp.Col),
						float64(flp.Row),
						float64(flp.Scale*0.75),
						color.RGBA{R: 0, G: 255, B: 0, A: 255},
					)
				}

				flp = flpc.LandmarkPoint(leftPupil, rightPupil, c.imgParams, perturbation, true)
				if flp.Row > 0 && flp.Col > 0 {
					c.DrawArc(float64(flp.Col),
						float64(flp.Row),
						float64(flp.Scale*0.75),
						color.RGBA{R: 0, G: 255, B: 0, A: 255},
					)
				}
			}
		}

		for _, mouth := range mouthCascades {
			for _, flpc := range c.flpcs[mouth] {
				flp := flpc.LandmarkPoint(leftPupil, rightPupil, c.imgParams, perturbation, false)
				if flp.Row > 0 && flp.Col > 0 {
					c.DrawArc(float64(flp.Col),
						float64(flp.Row),
						float64(flp.Scale*0.75),
						color.RGBA{R: 0, G: 255, B: 0, A: 255},
					)
				}
			}
		}
		flp := c.flpcs["lp84"][0].LandmarkPoint(leftPupil, rightPupil, c.imgParams, perturbation, true)
		if flp.Row > 0 && flp.Col > 0 {
			c.DrawArc(float64(flp.Col),
				float64(flp.Row),
				float64(flp.Scale*0.75),
				color.RGBA{R: 0, G: 255, B: 0, A: 255},
			)
		}
	}
}

func (c *Constructor) detectLeftPupil(face pigo.Detection) *pigo.Puploc {
	puploc := &pigo.Puploc{
		Row:      face.Row - int(0.085*float32(face.Scale)),
		Col:      face.Col - int(0.185*float32(face.Scale)),
		Scale:    float32(face.Scale) * 0.25,
		Perturbs: perturbation,
	}

	det := c.puploc.RunClassifier(puploc, c.imgParams, 0.0, false)
	if det.Row > 0 && det.Col > 0 {
		if ok := validPuplocDet(det, face); ok {
			return det
		}
	}

	return nil
}

func (c *Constructor) detectRightPupil(face pigo.Detection) *pigo.Puploc {
	puploc := &pigo.Puploc{
		Row:      face.Row - int(0.075*float32(face.Scale)),
		Col:      face.Col + int(0.185*float32(face.Scale)),
		Scale:    float32(face.Scale) * 0.25,
		Perturbs: perturbation,
	}

	det := c.puploc.RunClassifier(puploc, c.imgParams, 0.0, false)
	if det.Row > 0 && det.Col > 0 {
		if ok := validPuplocDet(det, face); ok {
			return det
		}
	}

	return nil
}

func (c *Constructor) DrawRect(x, y, w, h float64, color color.RGBA) {
	c.ImgCtx.DrawRectangle(x, y, w, h)
	c.ImgCtx.SetLineWidth(2.0)
	c.ImgCtx.SetStrokeStyle(gg.NewSolidPattern(color))
	c.ImgCtx.Stroke()
}

func (c *Constructor) DrawArc(x, y, w float64, color color.RGBA) {
	c.ImgCtx.DrawArc(x, y, w*0.15, 0, 2*math.Pi)
	c.ImgCtx.SetFillStyle(gg.NewSolidPattern(color))
	c.ImgCtx.Fill()
}

// stabilizeFace function is meant to smoothen the jittering
// as a side effect of how Pigo detect faces, ie.
// it compares the underlying pixel values, which means that small
// perturbation can occur even if the frame is almost stationary.
func stabilizeFace(face *faceTracker, det pigo.Detection) {
	const deadZone = 2.0
	alpha := 0.3

	x := float64(det.Col)
	y := float64(det.Row)
	scale := float64(det.Scale)

	if !face.detected {
		face.x, face.y = x, y
		face.scale = scale
		face.detected = true
		return
	}

	dx := x - face.x
	dy := y - face.y
	dist := math.Sqrt(dx*dx + dy*dy)

	// Real movement: follow much faster.
	if dist > face.scale*0.15 {
		alpha = 0.7
	}

	if math.Abs(dx) > deadZone {
		face.x += alpha * dx
	}

	if math.Abs(dy) > deadZone {
		face.y += alpha * dy
	}

	if math.Abs(scale-face.scale) > deadZone {
		face.scale += alpha * (scale - face.scale)
	}
}

// validPuplocDet sanitize the pupil detection and validates
// its coordinates against the bounding box of the detected face.
func validPuplocDet(det *pigo.Puploc, face pigo.Detection) bool {
	minRow := face.Row - face.Scale/2
	// Constraints the pupil localization to the upper portion of the face.
	maxRow := face.Row + face.Scale/10

	minCol := face.Col - face.Scale/2
	maxCol := face.Col + face.Scale/2

	if det.Row < minRow || det.Row > maxRow ||
		det.Col < minCol || det.Col > maxCol {
		return false
	}

	return true
}
