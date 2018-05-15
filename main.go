package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"unsafe"
)

type Pigo struct {
}

func NewPigo() *Pigo {
	return &Pigo{}
}

func main() {
	cascadeFile, err := ioutil.ReadFile("data/facefinder")
	if err != nil {
		log.Fatalf("Error reading the cascade file: %s", err)
	}

	pigo := NewPigo()
	pigo.Unpack(cascadeFile)
}

func (pg *Pigo) Unpack(packet []byte) {
	// We skip the first 8 bytes of the cascade file.
	var (
		pos int = 8
		treeCodes []int8
		treePredictions []float32
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
		treeDepth := binary.LittleEndian.Uint32(packet[pos:])
		fmt.Println("Tree depth: ", treeDepth)
		pos += 4

		// Get the number of cascade trees as 32-bit unsigned integer and write it into the buffer array.
		_, err := dataView.Write([]byte{packet[pos+0], packet[pos+1], packet[pos+2], packet[pos+3]})
		if err != nil {
			log.Fatalf("Error writing cascade trees into the buffer array: %v", err)
		}

		treeNum := binary.LittleEndian.Uint32(packet[pos:])
		pos += 4
		fmt.Println("Tree numbers: ", treeNum)

		for t := 0; t < int(treeNum); t++ {
			treeCodes = append(treeCodes, []int8{0, 0, 0, 0}...)

			res := packet[pos:pos+int(4*math.Pow(2, float64(treeDepth))-4)]
			signedPacket := *(*[]int8)(unsafe.Pointer(&res))
			treeCodes = append(treeCodes, signedPacket...)

			pos = pos + int(4 * math.Pow(2, float64(treeDepth))-4)

			// Read prediction from tree's leaf nodes.
			for i := 0; i < int(math.Pow(2, float64(treeDepth))); i++ {
				_, err := dataView.Write([]byte{packet[pos+0], packet[pos+1], packet[pos+2], packet[pos+3]})
				if err != nil {
					log.Fatalf("Error writing leaf node predictions into the buffer array: %v", err)
				}
				u32pred := binary.LittleEndian.Uint32(packet[pos:])
				// Convert uint32 to float32
				f32pred := math.Float32frombits(u32pred)

				treePredictions = append(treePredictions, f32pred)
				pos += 4
			}

			// Read tree nodes threshold values.
			_, err := dataView.Write([]byte{packet[pos+0], packet[pos+1], packet[pos+2], packet[pos+3]})
			if err != nil {
				log.Fatalf("Error writing tree nodes threshold value into the buffer array: %v", err)
			}
			u32thr := binary.LittleEndian.Uint32(packet[pos:])
			// Convert uint32 to float32
			f32thr := math.Float32frombits(u32thr)
			treeThreshold = append(treeThreshold, f32thr)

			pos += 4
		}
	}
}
