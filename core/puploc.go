package pigo

import (
	"bytes"
	"encoding/binary"
	"math"
	"math/rand"
	"sort"
	"unsafe"
)

// Puploc contains all the information resulted from the pupil detection
// needed for accessing from a global scope.
type Puploc struct {
	Row      int
	Col      int
	Scale    float32
	Perturbs int
}

// PuplocCascade is a general struct for storing
// the cascade tree values encoded into the binary file.
type PuplocCascade struct {
	Stages    uint32
	Scales    float32
	Trees     uint32
	TreeDepth uint32
	TreeCodes []int8
	TreePreds []float32
}

// UnpackCascade unpacks the pupil localization cascade file.
func (plc *PuplocCascade) UnpackCascade(packet []byte) (*PuplocCascade, error) {
	var (
		stages    uint32
		scales    float32
		trees     uint32
		treeDepth uint32
		treeCodes []int8
		treePreds []float32
	)

	pos := 0
	buff := make([]byte, 4)
	dataView := bytes.NewBuffer(buff)

	// Read the depth (size) of each tree and write it into the buffer array.
	_, err := dataView.Write([]byte{packet[pos+0], packet[pos+1], packet[pos+2], packet[pos+3]})
	if err != nil {
		return nil, err
	}

	if dataView.Len() > 0 {
		// Get the number of stages as 32-bit uint and write it into the buffer array.
		stages = binary.LittleEndian.Uint32(packet[pos:])
		_, err := dataView.Write([]byte{packet[pos+0], packet[pos+1], packet[pos+2], packet[pos+3]})
		if err != nil {
			return nil, err
		}
		pos += 4

		// Obtain the scale multiplier (applied after each stage) and write it into the buffer array.
		u32scales := binary.LittleEndian.Uint32(packet[pos:])
		// Convert uint32 to float32
		scales = *(*float32)(unsafe.Pointer(&u32scales))
		_, err = dataView.Write([]byte{packet[pos+0], packet[pos+1], packet[pos+2], packet[pos+3]})
		if err != nil {
			return nil, err
		}
		pos += 4

		// Obtain the number of trees per stage and write it into the buffer array.
		trees = binary.LittleEndian.Uint32(packet[pos:])
		_, err = dataView.Write([]byte{packet[pos+0], packet[pos+1], packet[pos+2], packet[pos+3]})
		if err != nil {
			return nil, err
		}
		pos += 4

		// Obtain the depth of each tree and write it into the buffer array.
		treeDepth = binary.LittleEndian.Uint32(packet[pos:])
		_, err = dataView.Write([]byte{packet[pos+0], packet[pos+1], packet[pos+2], packet[pos+3]})
		if err != nil {
			return nil, err
		}
		pos += 4

		// Traverse all the stages of the binary tree
		for s := 0; s < int(stages); s++ {
			// Traverse the branches of each stage
			for t := 0; t < int(trees); t++ {
				treeCodes = append(treeCodes, []int8{0, 0, 0, 0}...)

				code := packet[pos : pos+int(4*math.Pow(2, float64(treeDepth))-4)]
				// Convert unsigned bytecodes to signed ones.
				i8code := *(*[]int8)(unsafe.Pointer(&code))
				treeCodes = append(treeCodes, i8code...)

				pos = pos + int(4*math.Pow(2, float64(treeDepth))-4)

				// Read prediction from tree's leaf nodes.
				for i := 0; i < int(math.Pow(2, float64(treeDepth))); i++ {
					for l := 0; l < 2; l++ {
						_, err := dataView.Write([]byte{packet[pos+0], packet[pos+1], packet[pos+2], packet[pos+3]})
						if err != nil {
							return nil, err
						}
						u32pred := binary.LittleEndian.Uint32(packet[pos:])
						// Convert uint32 to float32
						f32pred := *(*float32)(unsafe.Pointer(&u32pred))
						treePreds = append(treePreds, f32pred)
						pos += 4
					}
				}
			}

		}
	}

	return &PuplocCascade{
		Stages:    stages,
		Scales:    scales,
		Trees:     trees,
		TreeDepth: treeDepth,
		TreeCodes: treeCodes,
		TreePreds: treePreds,
	}, nil
}

// RunDetector runs the pupil localization function.
func (plc *PuplocCascade) RunDetector(pl Puploc, img ImageParams) *Puploc {
	localization := func(r, c, s int, pixels []uint8, nrows, ncols, dim int) []int {
		root := 0
		pTree := int(math.Pow(2, float64(plc.TreeDepth)))

		for i := 0; i < int(plc.Stages); i++ {
			var dr, dc float32 = 0.0, 0.0

			for j := 0; j < int(plc.Trees); j++ {
				idx := 0
				for k := 0; k < int(plc.TreeDepth); k++ {
					r1 := min(nrows-1, max(0, (256*r+int(plc.TreeCodes[root+4*idx+0])*s)>>8))
					c1 := min(ncols-1, max(0, (256*r+int(plc.TreeCodes[root+4*idx+1])*s)>>8))
					r2 := min(nrows-1, max(0, (256*r+int(plc.TreeCodes[root+4*idx+2])*s)>>8))
					c2 := min(ncols-1, max(0, (256*r+int(plc.TreeCodes[root+4*idx+3])*s)>>8))

					bintest := func(r1, r2 uint8) uint8 {
						if r1 > r2 {
							return 1
						}
						return 0
					}
					idx = 2*idx + 1 + int(bintest(pixels[r1*dim+c1], pixels[r2*dim+c2]))
				}
				lutIdx := 2 * (int(plc.Trees)*pTree*i + pTree*j + idx - (pTree - 1))

				dr += plc.TreePreds[lutIdx+0]
				dc += plc.TreePreds[lutIdx+1]

				root += 4*pTree - 4
			}

			r += int(dr) * s
			c += int(dc) * s
			s = int(float32(s) * plc.Scales)
		}
		return []int{r, c, s}
	}
	rows, cols, scale := []int{}, []int{}, []int{}

	for i := 0; i < pl.Perturbs; i++ {
		st := float32(pl.Scale) * (0.25 + rand.Float32())
		rt := float32(pl.Row) + float32(pl.Scale)*0.15*(0.5-rand.Float32())
		ct := float32(pl.Col) + float32(pl.Scale)*0.15*(0.5-rand.Float32())

		res := localization(int(rt), int(ct), int(st), img.Pixels, img.Rows, img.Cols, img.Dim)

		rows = append(rows, res[0])
		cols = append(cols, res[1])
		scale = append(scale, res[2])
	}

	// sorting the perturbations in ascendent order
	sort.Sort(plocDet(rows))
	sort.Sort(plocDet(cols))
	sort.Sort(plocDet(scale))

	// get the median value of the sorted perturbation results
	return &Puploc{
		Row:   rows[int(math.Round(float64(pl.Perturbs)/2))],
		Col:   cols[int(math.Round(float64(pl.Perturbs)/2))],
		Scale: float32(scale[int(math.Round(float64(pl.Perturbs)/2))]),
	}
}

// min returns the minum value between two numbers
func min(val1, val2 int) int {
	if val1 < val2 {
		return val1
	}
	return val2
}

// max returns the maximum value between two numbers
func max(val1, val2 int) int {
	if val1 > val2 {
		return val1
	}
	return val2
}

// Implement custom sorting function on detection values.
type plocDet []int

func (q plocDet) Len() int      { return len(q) }
func (q plocDet) Swap(i, j int) { q[i], q[j] = q[j], q[i] }
func (q plocDet) Less(i, j int) bool {
	if q[i] < q[j] {
		return true
	}
	if q[i] > q[j] {
		return false
	}
	return q[i] < q[j]
}
