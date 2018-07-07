package pigo

import (
	"testing"
	"io/ioutil"
)

func TestUnpack(t *testing.T) {
	pigo := NewPigo()

	cascadeFile, err := ioutil.ReadFile("../data/facefinder")
	if err != nil {
		t.Fatalf("unable to read the cascade file: %v", err)
	}

	_, err = pigo.Unpack(cascadeFile)
	if err != nil {
		t.Fatalf("unable to unpack the face classification file: %v", err)
	}
}