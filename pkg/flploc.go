package pigo

import (
	"embed"
	"fmt"
	"math"
	"sync"
)

//go:embed cascade/lps/*
var fs embed.FS

// Because the embed directive does not support relative and dynamic file paths,
// the path must be known at compile time. That's why the cascade files
// should be included inside the parent directory.
const lpsDir = "cascade/lps"

// FlpCascade holds the binary representation of the facial landmark points cascade files.
type FlpCascade struct {
	*PuplocCascade
}

// We are using sync.Pool to avoid memory allocation on the heap
// in order to keep the GC overhead as small as possible.
var flplocPool = sync.Pool{
	New: func() any {
		return &Puploc{}
	},
}

// LandmarkPoint calculates the facial landmark point position based on the eyes localization results.
func (plc *PuplocCascade) LandmarkPoint(leftEye, rightEye *Puploc, img *ImageParams, perturb int, flipV bool) *Puploc {
	dx := (leftEye.Row - rightEye.Row) * (leftEye.Row - rightEye.Row)
	dy := (leftEye.Col - rightEye.Col) * (leftEye.Col - rightEye.Col)

	dist := math.Sqrt(float64(dx + dy))

	row := float64(leftEye.Row+rightEye.Row)/2.0 + 0.25*dist
	col := float64(leftEye.Col+rightEye.Col)/2.0 + 0.15*dist
	scale := 3.0 * dist

	flploc := flplocPool.Get().(*Puploc)
	defer flplocPool.Put(flploc)

	flploc.Row = int(row)
	flploc.Col = int(col)
	flploc.Scale = float32(scale)
	flploc.Perturbs = perturb

	if flipV {
		return plc.RunClassifier(flploc, img, 0.0, true)
	}

	return plc.RunClassifier(flploc, img, 0.0, false)
}

// ReadFlpCascadeDir reads the directory of the embedded
// facial landmark points cascade files and extracts their content.
func ReadFlpCascadeDir() (map[string][]*FlpCascade, error) {
	cascadeFiles, err := fs.ReadDir(lpsDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read the embedded files directory: %w", err)
	}

	flpcs := make(map[string][]*FlpCascade, len(cascadeFiles))

	for _, f := range cascadeFiles {
		data, err := fs.ReadFile(lpsDir + "/" + f.Name())
		if err != nil {
			return nil, fmt.Errorf("cannot read the cascade file: %w", err)
		}

		flpc := unpack(data)
		flpcs[f.Name()] = append(flpcs[f.Name()], &FlpCascade{flpc})
	}

	return flpcs, err
}
