/*
Package triangle is an image processing library, which converts images to computer generated art using delaunay triangulation.

The package provides a command line interface, supporting various options for the output customization.
Check the supported commands by typing:

	$ triangle --help

Using Go interfaces the API can expose the result either as raster or vector type.

Example to generate triangulated image and output the result as raster type:

	package main

	import (
		"fmt"
		"github.com/esimov/triangle"
	)

	func main() {
		p := &triangle.Processor{
			// Initialize struct variables
		}

		img := &triangle.Image{*p}
		_, _, err = img.Draw(file, fq, func() {})

		if err != nil {
			fmt.Printf("Error on triangulation process: %s", err.Error())
		}
	}


Example to generate triangulated image and output the result to SVG:

	package main

	import (
		"fmt"
		"github.com/esimov/triangle"
	)

	func main() {
		p := &triangle.Processor{
			// Initialize struct variables
		}

		svg := &triangle.SVG{
			Title:         "Delaunay image triangulator",
			Lines:         []triangle.Line{},
			Description:   "Convert images to computer generated art using delaunay triangulation.",
			StrokeWidth:   p.StrokeWidth,
			StrokeLineCap: "round", //butt, round, square
			Processor:     *p,
		}
		_, _, err = svg.Draw(file, fq, func() {
			// Call the closure function
		})

		if err != nil {
			fmt.Printf("Error on triangulation process: %s", err.Error())
		}
	}

 */
package triangle
