package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/disintegration/imaging"
	pigo "github.com/esimov/pigo/core"
	"github.com/fogleman/gg"
	"golang.org/x/term"
)

const banner = `
┌─┐┬┌─┐┌─┐
├─┘││ ┬│ │
┴  ┴└─┘└─┘

Go (Golang) Face detection library.
    Version: %s

`

// pipeName is the file name that indicates stdin/stdout is being used.
const pipeName = "-"

const (
	// MarkerRectangle - use rectangle as face detection marker
	MarkerRectangle string = "rect"
	// MarkerCircle - use circle as face detection marker
	MarkerCircle string = "circle"
	// MarkerEllipse - use ellipse as face detection marker
	MarkerEllipse string = "ellipse"
)

// Version indicates the current build version.
var Version string

var (
	dc        *gg.Context
	fd        *faceDetector
	plc       *pigo.PuplocCascade
	flpcs     map[string][]*pigo.FlpCascade
	imgParams *pigo.ImageParams
)

var (
	eyeCascades   = []string{"lp46", "lp44", "lp42", "lp38", "lp312"}
	mouthCascades = []string{"lp93", "lp84", "lp82", "lp81"}
)

// faceDetector struct contains Pigo face detector general settings.
type faceDetector struct {
	angle        float64
	cascadeFile  string
	destination  string
	minSize      int
	maxSize      int
	shiftFactor  float64
	scaleFactor  float64
	iouThreshold float64
	puploc       string
	flploc       string
	markDetEyes  bool
}

// coord holds the detection coordinates
type coord struct {
	Row   int `json:"x,omitempty"`
	Col   int `json:"y,omitempty"`
	Scale int `json:"size,omitempty"`
}

// detection holds the detection points of the various detection types
type detection struct {
	FacePoints     coord   `json:"face,omitempty"`
	EyePoints      []coord `json:"eyes,omitempty"`
	LandmarkPoints []coord `json:"landmark_points,omitempty"`
}

func main() {
	var (
		// Flags
		source       = flag.String("in", pipeName, "Source image")
		destination  = flag.String("out", pipeName, "Destination image")
		cascadeFile  = flag.String("cf", "", "Cascade binary file")
		minSize      = flag.Int("min", 20, "Minimum size of face")
		maxSize      = flag.Int("max", 1000, "Maximum size of face")
		shiftFactor  = flag.Float64("shift", 0.1, "Shift detection window by percentage")
		scaleFactor  = flag.Float64("scale", 1.1, "Scale detection window by percentage")
		angle        = flag.Float64("angle", 0.0, "0.0 is 0 radians and 1.0 is 2*pi radians")
		iouThreshold = flag.Float64("iou", 0.2, "Intersection over union (IoU) threshold")
		marker       = flag.String("marker", "rect", "Detection marker: rect|circle|ellipse")
		puploc       = flag.String("plc", "", "Pupils/eyes localization cascade file")
		flploc       = flag.String("flpc", "", "Facial landmark points cascade directory")
		markEyes     = flag.Bool("mark", true, "Mark detected eyes")
		jsonf        = flag.String("json", "", "Output the detection points into a json file")
	)

	log.SetFlags(0)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, fmt.Sprintf(banner, Version))
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(*source) == 0 || len(*cascadeFile) == 0 {
		log.Fatal("Usage: pigo -in input.jpg -out out.png -cf cascade/facefinder")
	}
	if *scaleFactor < 1.05 {
		log.Fatal("Scale factor must be greater than 1.05")
	}

	// Progress indicator
	s := new(spinner)
	s.start("Processing...")
	start := time.Now()

	fd = &faceDetector{
		angle:        *angle,
		destination:  *destination,
		cascadeFile:  *cascadeFile,
		minSize:      *minSize,
		maxSize:      *maxSize,
		shiftFactor:  *shiftFactor,
		scaleFactor:  *scaleFactor,
		iouThreshold: *iouThreshold,
		puploc:       *puploc,
		flploc:       *flploc,
		markDetEyes:  *markEyes,
	}

	var dst io.Writer
	if fd.destination != "empty" {
		if fd.destination == pipeName {
			if term.IsTerminal(int(os.Stdout.Fd())) {
				log.Fatalln("`-` should be used with a pipe for stdout")
			}
			dst = os.Stdout
		} else {
			fileTypes := []string{".jpg", ".jpeg", ".png"}
			ext := filepath.Ext(fd.destination)

			if !inSlice(ext, fileTypes) {
				log.Fatalf("Output file type not supported: %v", ext)
			}

			fn, err := os.OpenFile(fd.destination, os.O_CREATE|os.O_WRONLY, 0755)
			if err != nil {
				log.Fatalf("Unable to open output file: %v", err)
			}
			defer fn.Close()
			dst = fn
		}
	}

	faces, err := fd.detectFaces(*source)
	if err != nil {
		log.Fatalf("Detection error: %v", err)
	}

	dets, err := fd.drawFaces(faces, *marker)
	if err != nil {
		log.Fatalf("Error creating the image output: %s", err)
	}

	if fd.destination != "empty" {
		if err := fd.encodeImage(dst); err != nil {
			log.Fatalf("Error encoding the output image: %v", err)
		}
	}

	if *jsonf != "" {
		var out io.Writer
		if *jsonf == pipeName {
			out = os.Stdout
		} else {
			f, err := os.Create(*jsonf)
			defer f.Close()
			if err != nil {
				log.Fatalf("Could not create the json file: %s", err)
			}
			out = f
		}
		if err := json.NewEncoder(out).Encode(dets); err != nil {
			log.Fatalf("Error encoding the json file: %s", err)
		}
	}
	s.stop()
	log.Printf("\nDone in: \x1b[92m%.2fs\n", time.Since(start).Seconds())
}

// detectFaces run the detection algorithm over the provided source image.
func (fd *faceDetector) detectFaces(source string) ([]pigo.Detection, error) {
	var srcFile io.Reader
	if source == pipeName {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			log.Fatalln("`-` should be used with a pipe for stdin")
		}
		srcFile = os.Stdin
	} else {
		file, err := os.Open(source)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		srcFile = file
	}

	src, err := pigo.DecodeImage(srcFile)
	if err != nil {
		return nil, err
	}

	pixels := pigo.RgbToGrayscale(src)
	cols, rows := src.Bounds().Max.X, src.Bounds().Max.Y

	dc = gg.NewContext(cols, rows)
	dc.DrawImage(src, 0, 0)

	imgParams = &pigo.ImageParams{
		Pixels: pixels,
		Rows:   rows,
		Cols:   cols,
		Dim:    cols,
	}

	cParams := pigo.CascadeParams{
		MinSize:     fd.minSize,
		MaxSize:     fd.maxSize,
		ShiftFactor: fd.shiftFactor,
		ScaleFactor: fd.scaleFactor,
		ImageParams: *imgParams,
	}

	cascadeFile, err := ioutil.ReadFile(fd.cascadeFile)
	if err != nil {
		return nil, err
	}

	p := pigo.NewPigo()
	// Unpack the binary file. This will return the number of cascade trees,
	// the tree depth, the threshold and the prediction from tree's leaf nodes.
	classifier, err := p.Unpack(cascadeFile)
	if err != nil {
		return nil, err
	}

	if len(fd.puploc) > 0 {
		pl := pigo.NewPuplocCascade()
		cascade, err := ioutil.ReadFile(fd.puploc)
		if err != nil {
			return nil, err
		}
		plc, err = pl.UnpackCascade(cascade)
		if err != nil {
			return nil, err
		}

		if len(fd.flploc) > 0 {
			flpcs, err = pl.ReadCascadeDir(fd.flploc)
			if err != nil {
				return nil, err
			}
		}
	}

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	faces := classifier.RunCascade(cParams, fd.angle)

	// Calculate the intersection over union (IoU) of two clusters.
	faces = classifier.ClusterDetections(faces, fd.iouThreshold)

	return faces, nil
}

// drawFaces marks the detected faces with the marker type defined as parameter (rectangle|circle|ellipse).
func (fd *faceDetector) drawFaces(faces []pigo.Detection, marker string) ([]detection, error) {
	var (
		qThresh float32 = 5.0
		perturb         = 63
	)

	var (
		detections     []detection
		eyesCoords     []coord
		landmarkCoords []coord
		puploc         *pigo.Puploc
	)

	for _, face := range faces {
		if face.Q > qThresh {
			switch marker {
			case "rect":
				dc.DrawRectangle(float64(face.Col-face.Scale/2),
					float64(face.Row-face.Scale/2),
					float64(face.Scale),
					float64(face.Scale),
				)
			case "circle":
				dc.DrawArc(
					float64(face.Col),
					float64(face.Row),
					float64(face.Scale/2),
					0,
					2*math.Pi,
				)
			case "ellipse":
				dc.DrawEllipse(
					float64(face.Col),
					float64(face.Row),
					float64(face.Scale)/2,
					float64(face.Scale)/1.6,
				)
			}
			faceCoord := &coord{
				Col:   face.Row - face.Scale/2,
				Row:   face.Col - face.Scale/2,
				Scale: face.Scale,
			}

			dc.SetLineWidth(2.0)
			dc.SetStrokeStyle(gg.NewSolidPattern(color.RGBA{R: 255, G: 0, B: 0, A: 255}))
			dc.Stroke()

			if len(fd.puploc) > 0 && face.Scale > 50 {
				rect := image.Rect(
					face.Col-face.Scale/2,
					face.Row-face.Scale/2,
					face.Col+face.Scale/2,
					face.Row+face.Scale/2,
				)
				rows, cols := rect.Max.X-rect.Min.X, rect.Max.Y-rect.Min.Y
				ctx := gg.NewContext(rows, cols)
				faceZone := ctx.Image()

				// left eye
				puploc = &pigo.Puploc{
					Row:      face.Row - int(0.075*float32(face.Scale)),
					Col:      face.Col - int(0.175*float32(face.Scale)),
					Scale:    float32(face.Scale) * 0.25,
					Perturbs: perturb,
				}
				leftEye := plc.RunDetector(*puploc, *imgParams, fd.angle, false)
				if leftEye.Row > 0 && leftEye.Col > 0 {
					if fd.angle > 0 {
						drawEyeDetectionMarker(ctx,
							float64(cols/2-(face.Col-leftEye.Col)),
							float64(rows/2-(face.Row-leftEye.Row)),
							float64(leftEye.Scale),
							color.RGBA{R: 255, G: 0, B: 0, A: 255},
							fd.markDetEyes,
						)
						angle := (fd.angle * 180) / math.Pi
						rotated := imaging.Rotate(faceZone, 2*angle, color.Transparent)
						final := imaging.FlipH(rotated)

						dc.DrawImage(final, face.Col-face.Scale/2, face.Row-face.Scale/2)
					} else {
						drawEyeDetectionMarker(dc,
							float64(leftEye.Col),
							float64(leftEye.Row),
							float64(leftEye.Scale),
							color.RGBA{R: 255, G: 0, B: 0, A: 255},
							fd.markDetEyes,
						)
					}
					eyesCoords = append(eyesCoords, coord{
						Col:   leftEye.Row,
						Row:   leftEye.Col,
						Scale: int(leftEye.Scale),
					})
				}

				// right eye
				puploc = &pigo.Puploc{
					Row:      face.Row - int(0.075*float32(face.Scale)),
					Col:      face.Col + int(0.185*float32(face.Scale)),
					Scale:    float32(face.Scale) * 0.25,
					Perturbs: perturb,
				}

				rightEye := plc.RunDetector(*puploc, *imgParams, fd.angle, false)
				if rightEye.Row > 0 && rightEye.Col > 0 {
					if fd.angle > 0 {
						drawEyeDetectionMarker(ctx,
							float64(cols/2-(face.Col-rightEye.Col)),
							float64(rows/2-(face.Row-rightEye.Row)),
							float64(rightEye.Scale),
							color.RGBA{R: 255, G: 0, B: 0, A: 255},
							fd.markDetEyes,
						)
						// convert radians to angle
						angle := (fd.angle * 180) / math.Pi
						rotated := imaging.Rotate(faceZone, 2*angle, color.Transparent)
						final := imaging.FlipH(rotated)

						dc.DrawImage(final, face.Col-face.Scale/2, face.Row-face.Scale/2)
					} else {
						drawEyeDetectionMarker(dc,
							float64(rightEye.Col),
							float64(rightEye.Row),
							float64(rightEye.Scale),
							color.RGBA{R: 255, G: 0, B: 0, A: 255},
							fd.markDetEyes,
						)
					}
					eyesCoords = append(eyesCoords, coord{
						Col:   rightEye.Row,
						Row:   rightEye.Col,
						Scale: int(rightEye.Scale),
					})
				}

				if len(fd.flploc) > 0 {
					for _, eye := range eyeCascades {
						for _, flpc := range flpcs[eye] {
							flp := flpc.FindLandmarkPoints(leftEye, rightEye, *imgParams, perturb, false)
							if flp.Row > 0 && flp.Col > 0 {
								drawEyeDetectionMarker(dc,
									float64(flp.Col),
									float64(flp.Row),
									float64(flp.Scale*0.5),
									color.RGBA{R: 0, G: 0, B: 255, A: 255},
									false,
								)
								landmarkCoords = append(landmarkCoords, coord{
									Col:   flp.Row,
									Row:   flp.Col,
									Scale: int(flp.Scale),
								})
							}

							flp = flpc.FindLandmarkPoints(leftEye, rightEye, *imgParams, perturb, true)
							if flp.Row > 0 && flp.Col > 0 {
								drawEyeDetectionMarker(dc,
									float64(flp.Col),
									float64(flp.Row),
									float64(flp.Scale*0.5),
									color.RGBA{R: 0, G: 0, B: 255, A: 255},
									false,
								)
								landmarkCoords = append(landmarkCoords, coord{
									Col:   flp.Row,
									Row:   flp.Col,
									Scale: int(flp.Scale),
								})
							}
						}
					}

					for _, mouth := range mouthCascades {
						for _, flpc := range flpcs[mouth] {
							flp := flpc.FindLandmarkPoints(leftEye, rightEye, *imgParams, perturb, false)
							if flp.Row > 0 && flp.Col > 0 {
								drawEyeDetectionMarker(dc,
									float64(flp.Col),
									float64(flp.Row),
									float64(flp.Scale*0.5),
									color.RGBA{R: 0, G: 0, B: 255, A: 255},
									false,
								)
								landmarkCoords = append(landmarkCoords, coord{
									Col:   flp.Row,
									Row:   flp.Col,
									Scale: int(flp.Scale),
								})
							}
						}
					}
					flp := flpcs["lp84"][0].FindLandmarkPoints(leftEye, rightEye, *imgParams, perturb, true)
					if flp.Row > 0 && flp.Col > 0 {
						drawEyeDetectionMarker(dc,
							float64(flp.Col),
							float64(flp.Row),
							float64(flp.Scale*0.5),
							color.RGBA{R: 0, G: 0, B: 255, A: 255},
							false,
						)
						landmarkCoords = append(landmarkCoords, coord{
							Col:   flp.Row,
							Row:   flp.Col,
							Scale: int(flp.Scale),
						})
					}
				}
			}
			detections = append(detections, detection{
				FacePoints:     *faceCoord,
				EyePoints:      eyesCoords,
				LandmarkPoints: landmarkCoords,
			})
		}
	}
	return detections, nil
}

func (fd *faceDetector) encodeImage(dst io.Writer) error {
	var err error
	img := dc.Image()

	switch dst.(type) {
	case *os.File:
		ext := filepath.Ext(dst.(*os.File).Name())
		switch ext {
		case "", ".jpg", ".jpeg":
			err = jpeg.Encode(dst, img, &jpeg.Options{Quality: 100})
		case ".png":
			err = png.Encode(dst, img)
		default:
			err = errors.New("unsupported image format")
		}
	default:
		err = jpeg.Encode(dst, img, &jpeg.Options{Quality: 100})
	}
	return err
}

type spinner struct {
	done chan struct{}
}

// Start process
func (s *spinner) start(message string) {
	s.done = make(chan struct{}, 1)

	go func() {
		for {
			for _, r := range `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏` {
				select {
				case <-s.done:
					return
				default:
					fmt.Fprintf(os.Stderr, "\r%s%s %c%s", message, "\x1b[35m", r, "\x1b[39m")
					time.Sleep(time.Millisecond * 100)
				}
			}
		}
	}()
}

// End process
func (s *spinner) stop() {
	s.done <- struct{}{}
}

// inSlice checks if the item exists in the slice.
func inSlice(item string, slice []string) bool {
	for _, it := range slice {
		if it == item {
			return true
		}
	}
	return false
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
