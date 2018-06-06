# ![pigo](https://user-images.githubusercontent.com/883386/40915591-525ae70a-6805-11e8-8991-5841d1270298.png)

[![Build Status](https://travis-ci.org/esimov/pigo.svg?branch=master)](https://travis-ci.org/esimov/pigo)
[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://godoc.org/github.com/esimov/pigo/core)
[![license](https://img.shields.io/github/license/mashape/apistatus.svg?style=flat)](./LICENSE)
[![release](https://img.shields.io/badge/release-v1.0.1-blue.svg)]()

Pigo is a face detection library implemented in Go based on ***Pixel Intensity Comparison-based Object detection*** paper (https://arxiv.org/pdf/1305.4537.pdf). 

| Rectangle face marker | Circle face marker
|:--:|:--:
| ![rectangle](https://user-images.githubusercontent.com/883386/40916662-2fbbae1a-6809-11e8-8afd-d4ed40c7d4e9.png) | ![circle](https://user-images.githubusercontent.com/883386/40916683-447088a8-6809-11e8-942f-3112c10bede3.png) |

### Motivation
I've intended to implement this face detection method in Go, since the only existing solution for face detection in the Go ecosystem is using bindings to OpenCV, but installing OpenCV on various platforms is sometimes daunting. 

This library does not require any third party modules to be installed. However in case you wish to try the real time, webcam based face detection you might need to have Python2 and OpenCV installed, but the core API does not require any third party and external modules. 

Since I haven't found any viable existing solution for accessing webcam in Go, Python is used for capturing the webcam and transferring the binary data to Go through `exec.CommandContext` method.

### Key features
- [x] High processing speed.
- [x] There is no need for image preprocessing prior to detection.
- [x] There is no need for the computation of integral images, image pyramid, HOG pyramid or any other similar data structure.
- [x] The face detection is based on pixel intensity comparison encoded in the binary file dat tree structure.

### Todo
- [ ] Object rotation detection.

## Install
Install Go, set your `GOPATH`, and make sure `$GOPATH/bin` is on your `PATH`.

```bash
$ export GOPATH="$HOME/go"
$ export PATH="$PATH:$GOPATH/bin"
```
Next download the project and build the binary file.

```bash
$ go get -u -f github.com/esimov/pigo/cmd/pigo
$ go install
```
### Binary releases
Also you can obtain the generated binary files in the [releases](https://github.com/esimov/pigo/releases) folder in case you do not have installed or do not want to install Go.

## API
Below is a minimal example of using the face detection API. 

First you need to load and parse the binary classifier, then convert the image to grayscale mode, 
and finally to run the cascade function which returns a slice containing the row, column, scale and the detection score.

```Go
cascadeFile, err := ioutil.ReadFile("/path/to/cascade/file")
if err != nil {
	log.Fatalf("Error reading the cascade file: %v", err)
}

src, err := pigo.GetImage("/path/to/image")
if err != nil {
	log.Fatalf("Cannot open the image file: %v", err)
}

sampleImg := pigo.RgbToGrayscale(src)

cParams := pigo.CascadeParams{
	MinSize:     1000,
	MaxSize:     20,
	ShiftFactor: 0.1,
	ScaleFactor: 1.1,
}
cols, rows := src.Bounds().Max.X, src.Bounds().Max.Y
imgParams := pigo.ImageParams{sampleImg, rows, cols, cols}

pigo := pigo.NewPigo()
// Unpack the binary file. This will return the number of cascade trees,
// the tree depth, the threshold and the prediction from tree's leaf nodes.
classifier := pigo.Unpack(cascadeFile)

// Run the classifier over the obtained leaf nodes and return the detection results.
// The result contains quadruplets representing the row, column, scale and detection score.
dets := classifier.RunCascade(imgParams, cParams)

// Calculate the intersection over union (IoU) of two clusters.
dets = classifier.ClusterDetections(dets, 0.2)
```

## Usage
A command line utility is bundled into the library to facilitate face detection in static images.

```bash
$ pigo -in input.jpg -out out.jpg -cf data/facefinder
```

### Supported flags:

```bash
$ pigo --help
┌─┐┬┌─┐┌─┐
├─┘││ ┬│ │
┴  ┴└─┘└─┘

Go (Golang) Face detection library.
    Version: 1.0.1

  -cf string
    	Cascade binary file
  -circle
    	Use circle as detection marker
  -in string
    	Source image
  -iou float
    	Intersection over union (IoU) threshold (default 0.2)
  -max int
    	Maximum size of face (default 1000)
  -min int
    	Minimum size of face (default 20)
  -out string
    	Destination image
  -scale float
    	Scale detection window by percentage (default 1.1)
  -shift float
    	Shift detection window by percentage (default 0.1)

```

### Real time face detection

In case you want to test the library real time face detection capabilities using a webcam there is an example included in the `webcam` folder. Prior to run it you need to have Pyton2 and OpenCV2 installed. In order to run it select the `webcam` folder and type:

```bash
$ go run main.go -cf "../data/facefinder"
```
Then access the `http://localhost:8081/cam` url from a web browser.


#### Other implementation

https://github.com/tehnokv/picojs

## Author
Simo Endre [@simo_endre](https://twitter.com/simo_endre)

## License

Copyright © 2018 Simo Endre

This project is under the MIT License. See the LICENSE file for the full license text.

