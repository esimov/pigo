package pigo

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"log"
	"math"
	"unsafe"
)

type pigo struct {
	treeDepth     uint32
	treeNum       uint32
	treeCodes     []int8
	treePred      []float32
	treeThreshold []float32
}

func NewPigo() *pigo {
	return &pigo{}
}

// Unpack unpack the binary face classification file.
func (pg *pigo) Unpack(packet []byte) *pigo {
	// We skip the first 8 bytes of the cascade file.
	var (
		pos           int = 8
		treeDepth     uint32
		treeNum       uint32
		treeCodes     []int8
		treePred      []float32
		treeThreshold []float32
	)

	buff := make([]byte, 4)
	dataView := bytes.NewBuffer(buff)

	// Read the depth (size) of each tree and write it into the buffer array.
	_, err := dataView.Write([]byte{packet[pos+0], packet[pos+1], packet[pos+2], packet[pos+3]})
	if err != nil {
		log.Fatalf("Error writing tree size into the buffer array: %v", err)
	}

	if dataView.Len() > 0 {
		treeDepth = binary.LittleEndian.Uint32(packet[pos:])
		fmt.Println("Tree depth: ", treeDepth)
		pos += 4

		// Get the number of cascade trees as 32-bit unsigned integer and write it into the buffer array.
		_, err := dataView.Write([]byte{packet[pos+0], packet[pos+1], packet[pos+2], packet[pos+3]})
		if err != nil {
			log.Fatalf("Error writing cascade trees into the buffer array: %v", err)
		}

		treeNum = binary.LittleEndian.Uint32(packet[pos:])
		pos += 4
		fmt.Println("Tree numbers: ", treeNum)

		for t := 0; t < int(treeNum); t++ {
			treeCodes = append(treeCodes, []int8{0, 0, 0, 0}...)

			code := packet[pos : pos+int(4*math.Pow(2, float64(treeDepth))-4)]
			signedCode := *(*[]int8)(unsafe.Pointer(&code))
			treeCodes = append(treeCodes, signedCode...)

			pos = pos + int(4*math.Pow(2, float64(treeDepth))-4)

			// Read prediction from tree's leaf nodes.
			for i := 0; i < int(math.Pow(2, float64(treeDepth))); i++ {
				_, err := dataView.Write([]byte{packet[pos+0], packet[pos+1], packet[pos+2], packet[pos+3]})
				if err != nil {
					log.Fatalf("Error writing leaf node predictions into the buffer array: %v", err)
				}
				u32pred := binary.LittleEndian.Uint32(packet[pos:])
				// Convert uint32 to float32
				f32pred := *(*float32)(unsafe.Pointer(&u32pred))
				treePred = append(treePred, f32pred)
				pos += 4
			}

			// Read tree nodes threshold values.
			_, err := dataView.Write([]byte{packet[pos+0], packet[pos+1], packet[pos+2], packet[pos+3]})
			if err != nil {
				log.Fatalf("Error writing tree nodes threshold value into the buffer array: %v", err)
			}
			u32thr := binary.LittleEndian.Uint32(packet[pos:])
			// Convert uint32 to float32
			f32thr := *(*float32)(unsafe.Pointer(&u32thr))
			treeThreshold = append(treeThreshold, f32thr)
			pos += 4
		}
	}

	return &pigo{
		treeDepth,
		treeNum,
		treeCodes,
		treePred,
		treeThreshold,
	}
}

// classifyRegion constructs the classification function based on the parsed binary data.
func (pg *pigo) classifyRegion(row, col, center int, pixels []uint8, dim int) float32 {
	var (
		root  int
		out   float32
		pTree = int(math.Pow(2, float64(pg.treeDepth)))
	)

	row = row * 256
	col = col * 256

	for i := 0; i < int(pg.treeNum); i++ {
		var idx = 1
		for j := 0; j < int(pg.treeDepth); j++ {
			var pix int
			idx1 := ((row+int(pg.treeCodes[root+4*idx+0])*center)>>8)*dim + ((col + int(pg.treeCodes[root+4*idx+1])*center) >> 8)
			idx2 := ((row+int(pg.treeCodes[root+4*idx+2])*center)>>8)*dim + ((col + int(pg.treeCodes[root+4*idx+3])*center) >> 8)

			if pixels[idx1] <= pixels[idx2] {
				pix = 1
			} else {
				pix = 0
			}
			idx = 2*idx + pix
		}

		out += pg.treePred[pTree*i+idx-pTree]

		if out <= pg.treeThreshold[i] {
			return -1
		}
		root += 4 * pTree
	}
	fmt.Println(out - pg.treeThreshold[pg.treeNum-1]);
	return out - pg.treeThreshold[pg.treeNum-1]
}

type CascadeParams struct {
	MinSize     int
	MaxSize     int
	ShiftFactor float64
	ScaleFactor float64
}

type ImageParams struct {
	*image.NRGBA
	Rows int
	Cols int
	Dim  int
}

type detection struct {
	row    int
	col    int
	center int
	q      float32
}

func (pg *pigo) RunCascade(img ImageParams, opts CascadeParams) []detection {
	var detections []detection
	center := opts.MinSize

	// Run the classification function over the detection window
	// and check if the false positive rate is over a certain value.
	for center <= opts.MaxSize {
		step := int(math.Max(opts.ShiftFactor*float64(center), 1))
		offset := (center/2 + 1)

		for row := offset; row <= img.Rows-offset; row += step {
			for col := offset; col <= img.Cols-offset; col += step {
				q := pg.classifyRegion(row, col, center, img.Pix, img.Dim)
				//fmt.Println(q)
				if q > 0.0 {
					detections = append(detections, detection{row, col, center, q})
				}
			}
		}
		center *= int(float64(center) * opts.ScaleFactor)
	}
	return detections
}
