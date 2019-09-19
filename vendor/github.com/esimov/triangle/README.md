
# ![Triangle logo](https://user-images.githubusercontent.com/883386/32769128-4d9625c6-c923-11e7-9a96-030f2f0efff3.png)

[![Build Status](https://travis-ci.org/esimov/triangle.svg?branch=master)](https://travis-ci.org/esimov/triangle)
[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://godoc.org/github.com/esimov/triangle)
[![license](https://img.shields.io/github/license/mashape/apistatus.svg?style=flat)](./LICENSE)
[![release](https://img.shields.io/badge/release-v1.0.3-blue.svg)](https://github.com/esimov/triangle/releases/tag/v1.0.3)
[![homebrew](https://img.shields.io/badge/homebrew-v1.0.3-orange.svg)](https://github.com/esimov/homebrew-triangle)

Triangle is a tool to generate image arts with [delaunay triangulation](https://en.wikipedia.org/wiki/Delaunay_triangulation). It takes an input image and converts it to an abstract image composed of tiles of triangles.

![Sample image](https://github.com/esimov/triangle/blob/master/output/sample_3.png)

### The technique
* First the image is blured out to smothen the sharp pixel edges. The more blured an image is the more diffused the generated output will be.
* Second the resulted image is converted to grayscale mode. 
* Then a [sobel](https://en.wikipedia.org/wiki/Sobel_operator) filter operator is applied on the grayscaled image to obtain the image edges. An optional threshold value is applied to filter out the representative pixels of the resulting image.
* Lastly the delaunay algorithm is applied on the pixels obtained from the previous step.

```go
blur = tri.Stackblur(img, uint32(width), uint32(height), uint32(*blurRadius))
gray = tri.Grayscale(blur)
sobel = tri.SobelFilter(gray, float64(*sobelThreshold))
points = tri.GetEdgePoints(sobel, *pointsThreshold, *maxPoints)

triangles = delaunay.Init(width, height).Insert(points).GetTriangles()
```
## Installation and usage
```bash
$ go get -u -f github.com/esimov/triangle/cmd/triangle
$ go install
```
## MacOS (Brew) install
The library can be installed via Homebrew too or by downloading the binary file from the [releases](https://github.com/esimov/triangle/releases) folder.

```bash
$ brew tap esimov/triangle
$ brew install triangle
```

### Supported commands

```bash
$ triangle --help
```
The following flags are supported:

| Flag | Default | Description |
| --- | --- | --- |
| `in` | n/a | Input file |
| `out` | n/a | Output file |
| `blur` | 4 | Blur radius |
| `max` | 2500 | Maximum number of points |
| `noise` | 0 | Noise factor |
| `points` | 20 | Points threshold |
| `sobel` | 10 | Sobel filter threshold |
| `solid` | false | Solid line color |
| `wireframe` | 0 | Wireframe mode (without,with,both) |
| `stroke` | 1 | Stroke width |
| `gray` | false | Convert to grayscale |
| `web` | false | Output SVG in browser |

#### Output as image or SVG
By default the output is saved to an image file, but you can export the resulted vertices even to an SVG file. The CLI tool can recognize the output type directly from the file extension. This is a handy addition for those who wish to generate large images without guality loss.

```bash
$ triangle -in samples/input.jpg -out output.svg
```

Using with `-web` flag you can access the generated svg file directly on the web browser.


```bash
$ triangle -in samples/input.jpg -out output.svg -web=true
```

### Multiple image processing with a single command
You can transform even multiple images from a specific folder with a single command by declaring as `-in` flag the source folder and as `-out` flag the destination folder.

```bash
$ triangle -in ./samples/ -out ./ouput -wireframe=0 -max=3500 -stroke=2 -blur=2 -noise=4
```
### Tweaks
Setting a lower points threshold, the resulted image will be more like a cubic painting. You can even add a noise factor, generating a more artistic, grainy image.

Here are some examples you can experiment with:
```bash
$ triangle -in samples/input.jpg -out output.png -wireframe=0 -max=3500 -stroke=2 -blur=2
$ triangle -in samples/input.jpg -out output.png -wireframe=2 -max=5500 -stroke=1 -blur=10
```

### Examples

<a href="https://github.com/esimov/triangle/blob/master/output/sample_3.png"><img src="https://github.com/esimov/triangle/blob/master/output/sample_3.png" width=420/></a>
<a href="https://github.com/esimov/triangle/blob/master/output/sample_4.png"><img src="https://github.com/esimov/triangle/blob/master/output/sample_4.png" width=420/></a>
<a href="https://github.com/esimov/triangle/blob/master/output/sample_5.png"><img src="https://github.com/esimov/triangle/blob/master/output/sample_5.png" width=420/></a>
<a href="https://github.com/esimov/triangle/blob/master/output/sample_6.png"><img src="https://github.com/esimov/triangle/blob/master/output/sample_6.png" width=420/></a>
![Sample_0](https://github.com/esimov/triangle/blob/master/output/sample_0.png)
![Sample_1](https://github.com/esimov/triangle/blob/master/output/sample_1.png)
![Sample_11](https://github.com/esimov/triangle/blob/master/output/sample_11.png)
![Sample_8](https://github.com/esimov/triangle/blob/master/output/sample_8.png)


## License
Copyright © 2018 Endre Simo

This project is under the MIT License. See the [LICENSE](https://github.com/esimov/triangle/blob/master/LICENSE) file for the full license text.
