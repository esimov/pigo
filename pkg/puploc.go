package pigo

import (
	"embed"
	_ "embed"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"slices"
	"sync"
	"unsafe"
)

//go:embed cascade/puploc
var plcFs embed.FS

// Puploc is used to gather all the relevant information
// resulted from the pupil detection.
type Puploc struct {
	Row   int
	Col   int
	Scale float32
	// The detector might produce certain overlapping results,
	// due to the small perturbations in position and scale.
	// This value defines how many times a detection should be
	// performed over the quasi identical overlapping results.
	Perturbs int
}

// PuplocCascade is a general struct for storing
// the cascade tree values encoded into the binary file.
type PuplocCascade struct {
	treeCodes []int8
	treePreds []float32
	scales    float32
	stages    uint32
	trees     uint32
	treeDepth uint32
}

// UnpackPlc reads and unpack the embedded pupil localization (puploc) cascade file.
// Returns error in case of failure on reading the embedded file.
func UnpackPlc() (*PuplocCascade, error) {
	f, err := plcFs.ReadFile("cascade/puploc")
	if err != nil {
		return nil, fmt.Errorf("cannot read the embedded cascade file: %w", err)
	}

	return unpack(f), nil
}

// decode unpacks the cascade file provided as byte array and decodes the results to PuplocCascade.
func unpack(cascade []byte) *PuplocCascade {
	var (
		stages    uint32
		scales    float32
		trees     uint32
		treeDepth uint32

		treeCodes = make([]int8, 0, 409200)
		treePreds = make([]float32, 0, 204800)
	)

	pos := 0
	// Get the number of stages as 32-bit unsigned integer.
	stages = binary.LittleEndian.Uint32(cascade[pos:])
	pos += 4

	// Obtain the scale multiplier (applied after each stage).
	u32scales := binary.LittleEndian.Uint32(cascade[pos:])
	// Convert uint32 to float32
	scales = *(*float32)(unsafe.Pointer(&u32scales))
	pos += 4

	// Obtain the number of trees per stage.
	trees = binary.LittleEndian.Uint32(cascade[pos:])
	pos += 4

	// Obtain the depth of each tree.
	treeDepth = binary.LittleEndian.Uint32(cascade[pos:])
	pos += 4

	// Traverse all the stages of the binary tree.
	for s := 0; s < int(stages); s++ {
		// Traverse the branches of each stage.
		for t := 0; t < int(trees); t++ {
			depth := int(pow(2, int(treeDepth)))

			code := cascade[pos : pos+4*depth-4]
			// Convert unsigned bytecodes to signed ones.
			i8code := *(*[]int8)(unsafe.Pointer(&code))
			treeCodes = append(treeCodes, i8code...)

			pos += 4*depth - 4

			// Read prediction from tree's leaf nodes.
			for i := 0; i < depth; i++ {
				for l := 0; l < 2; l++ {
					u32pred := binary.LittleEndian.Uint32(cascade[pos:])
					// Convert uint32 to float32
					f32pred := *(*float32)(unsafe.Pointer(&u32pred))
					treePreds = append(treePreds, f32pred)
					pos += 4
				}
			}
		}

	}

	return &PuplocCascade{
		stages:    stages,
		scales:    scales,
		trees:     trees,
		treeDepth: treeDepth,
		treeCodes: treeCodes,
		treePreds: treePreds,
	}
}

// classifyRegion applies the face classification function over an image.
func (plc *PuplocCascade) classifyRegion(r, c, s float32, treeDepth, rows, cols int, pixels []uint8, dim int, flipV bool) []float32 {
	var (
		c1, c2 int
		root   int
	)

	for i := 0; i < int(plc.stages); i++ {
		var dr, dc float32 = 0.0, 0.0

		for j := 0; j < int(plc.trees); j++ {
			idx := 0
			for k := 0; k < int(plc.treeDepth); k++ {
				r1 := min(rows-1, max(0, (256*int(r)+int(plc.treeCodes[root+4*idx+0])*int(math.Round(float64(s))))>>8))
				r2 := min(rows-1, max(0, (256*int(r)+int(plc.treeCodes[root+4*idx+2])*int(math.Round(float64(s))))>>8))

				// flipV means that we wish to flip the column coordinates sign in the tree nodes.
				// This is required at running the facial landmark detector over the right side of the detected face.
				if flipV {
					c1 = min(cols-1, max(0, (256*int(c)+int(-plc.treeCodes[root+4*idx+1])*int(math.Round(float64(s))))>>8))
					c2 = min(cols-1, max(0, (256*int(c)+int(-plc.treeCodes[root+4*idx+3])*int(math.Round(float64(s))))>>8))
				} else {
					c1 = min(cols-1, max(0, (256*int(c)+int(plc.treeCodes[root+4*idx+1])*int(math.Round(float64(s))))>>8))
					c2 = min(cols-1, max(0, (256*int(c)+int(plc.treeCodes[root+4*idx+3])*int(math.Round(float64(s))))>>8))
				}

				binaryTest := func(p1, p2 uint8) uint8 {
					if p1 > p2 {
						return 1
					}
					return 0
				}
				idx = 2*idx + 1 + int(binaryTest(pixels[r1*dim+c1], pixels[r2*dim+c2]))
			}
			lutIdx := 2 * (int(plc.trees)*treeDepth*i + treeDepth*j + idx - (treeDepth - 1))

			dr += plc.treePreds[lutIdx+0]
			if flipV {
				dc += -plc.treePreds[lutIdx+1]
			} else {
				dc += plc.treePreds[lutIdx+1]
			}
			root += 4*treeDepth - 4
		}

		r += dr * s
		c += dc * s
		s *= plc.scales
	}

	return []float32{r, c, s}
}

// classifyRotatedRegion applies the face classification function over a rotated image.
func (plc *PuplocCascade) classifyRotatedRegion(r, c, s float32, a float64, treeDepth, rows, cols int, pixels []uint8, dim int, flipV bool) []float32 {
	var (
		row1, col1, row2, col2 int
		root                   int
	)

	qCosTable := [...]float32{256, 251, 236, 212, 181, 142, 97, 49, 0, -49, -97, -142, -181, -212, -236, -251, -256, -251, -236, -212, -181, -142, -97, -49, 0, 49, 97, 142, 181, 212, 236, 251, 256}
	qSinTable := [...]float32{0, 49, 97, 142, 181, 212, 236, 251, 256, 251, 236, 212, 181, 142, 97, 49, 0, -49, -97, -142, -181, -212, -236, -251, -256, -251, -236, -212, -181, -142, -97, -49, 0}

	qsin := s * qSinTable[int(32.0*a)] //s*(256.0*math.Sin(2*math.Pi*a))
	qcos := s * qCosTable[int(32.0*a)] //s*(256.0*math.Cos(2*math.Pi*a))

	for i := 0; i < int(plc.stages); i++ {
		var dr, dc float32 = 0.0, 0.0

		for j := 0; j < int(plc.trees); j++ {
			idx := 0
			for k := 0; k < int(plc.treeDepth); k++ {
				row1 = int(plc.treeCodes[root+4*idx+0])
				row2 = int(plc.treeCodes[root+4*idx+2])

				// flipV means that we wish to flip the column coordinates sign in the tree nodes.
				// This is required at running the facial landmark detector over the right side of the detected face.
				if flipV {
					col1 = int(-plc.treeCodes[root+4*idx+1])
					col2 = int(-plc.treeCodes[root+4*idx+3])
				} else {
					col1 = int(plc.treeCodes[root+4*idx+1])
					col2 = int(plc.treeCodes[root+4*idx+3])
				}

				r1 := min(rows-1, max(0, 65536*int(r)+int(qcos)*row1-int(qsin)*col1)>>16)
				c1 := min(cols-1, max(0, 65536*int(c)+int(qsin)*row1+int(qcos)*col1)>>16)
				r2 := min(rows-1, max(0, 65536*int(r)+int(qcos)*row2-int(qsin)*col2)>>16)
				c2 := min(cols-1, max(0, 65536*int(c)+int(qsin)*row2+int(qcos)*col2)>>16)

				binaryTest := func(px1, px2 uint8) int {
					if px1 <= px2 {
						return 1
					}
					return 0
				}
				idx = 2*idx + 1 + binaryTest(pixels[r1*dim+c1], pixels[r2*dim+c2])
			}
			lutIdx := 2 * (int(plc.trees)*treeDepth*i + treeDepth*j + idx - (treeDepth - 1))

			dr += plc.treePreds[lutIdx+0]
			if flipV {
				dc += -plc.treePreds[lutIdx+1]
			} else {
				dc += plc.treePreds[lutIdx+1]
			}

			root += 4*treeDepth - 4
		}

		r += dr * s
		c += dc * s
		s *= plc.scales
	}

	return []float32{r, c, s}
}

// puplocPool holds the pupil localization values in a sync pool.
// It's single purpose is to avoid creating new thread pools
// each time the detector is running, instead re-use the existing one.
type puplocPool struct {
	rows  []float32
	cols  []float32
	scale []float32
}

// Create a sync.Pool for further reusing the allocated memory space
// in order to keep the GC overhead as low as possible.
var plcPool = sync.Pool{
	New: func() any {
		return &puplocPool{
			rows:  make([]float32, 63),
			cols:  make([]float32, 63),
			scale: make([]float32, 63),
		}
	},
}

// RunClassifier runs the pupil localization function.
func (plc *PuplocCascade) RunClassifier(pl *Puploc, img ImageParams, faceAngle float64, flipV bool) *Puploc {
	var res []float32

	det := plcPool.Get().(*puplocPool)
	defer plcPool.Put(det)

	treeDepth := int(pow(2, int(plc.treeDepth)))

	for i := 0; i < pl.Perturbs; i++ {
		row := float32(pl.Row) + float32(pl.Scale)*0.15*(0.5-rand.Float32())
		col := float32(pl.Col) + float32(pl.Scale)*0.15*(0.5-rand.Float32())
		sc := float32(pl.Scale) * (0.925 + 0.15*rand.Float32())

		if faceAngle > 0.0 {
			if faceAngle > 1.0 {
				faceAngle = 1.0
			}
			res = plc.classifyRotatedRegion(row, col, sc, faceAngle, treeDepth, img.Rows, img.Cols, img.Pixels, img.Dim, flipV)
		} else {
			res = plc.classifyRegion(row, col, sc, treeDepth, img.Rows, img.Cols, img.Pixels, img.Dim, flipV)
		}

		det.rows[i] = res[0]
		det.cols[i] = res[1]
		det.scale[i] = res[2]
	}

	// Sorting the perturbations in ascendent order
	slices.Sort(det.rows)
	slices.Sort(det.cols)
	slices.Sort(det.scale)

	// Get the median value of the sorted perturbation results
	return &Puploc{
		Row:   int(det.rows[int(math.Round(float64(pl.Perturbs)/2))]),
		Col:   int(det.cols[int(math.Round(float64(pl.Perturbs)/2))]),
		Scale: det.scale[int(math.Round(float64(pl.Perturbs)/2))],
	}
}
