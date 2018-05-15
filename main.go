package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
)

type Pigo struct {
}

func NewPigo() *Pigo {
	return &Pigo{}
}

func main() {
	cascadeFile, err := ioutil.ReadFile("facefinder")
	if err != nil {
		log.Fatalf("Error reading the cascade file: %s", err)
	}

	fmt.Println(cascadeFile[0])

	pigo := NewPigo()
	pigo.Unpack(cascadeFile)
}

func (pg *Pigo) Unpack(packet []byte) {
	// We skip the first 8 bytes of the cascade file.
	var pos int = 8

	buff := make([]byte, 4)
	dataView := bytes.NewBuffer(buff)

	fmt.Println("LEN:", dataView.Len())
	_, err := dataView.Write([]byte{packet[pos+0], packet[pos+1], packet[pos+2], packet[pos+3]})
	if err != nil {
		log.Fatalf("Error writing buffer bytes: %v", err)
	}

	if dataView.Len() > 0 {
		// Read the depth of each tree.
		depthTree := binary.LittleEndian.Uint32(packet[pos:])
		fmt.Println("Depth tree: ", depthTree)

		pos += 4
		_, err := dataView.Write([]byte{packet[pos+0], packet[pos+1], packet[pos+2], packet[pos+3]})
		if err != nil {
			log.Fatalf("Error writing buffer bytes: %v", err)
		}
		numTree := binary.LittleEndian.Uint32(packet[pos:])
		fmt.Println("Num tree: ", numTree)
	}
}
