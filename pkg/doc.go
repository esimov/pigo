/*
Package pigo is a lightweight pure Go face detection, pupil/eyes localization and facial landmark points detection library
based on Pixel Intensity Comparison-based Object detection paper (https://arxiv.org/pdf/1305.4537.pdf).
Is platform agnostic and does not require any external dependencies and third party modules.

Face detection API example.

First the source image should be converted to grayscale mode, then the cascade function can be invoked, which returns a slice containing the row, column, the scale and the detection score.

	src, err := pigo.GetImage("/path/to/image")
	if err != nil {
		log.Fatalf("Cannot open the image file: %v", err)
	}

	pixels := pigo.RgbToGrayscale(src)
	cols, rows := src.Bounds().Max.X, src.Bounds().Max.Y

	cParams := pigo.CascadeParams{
		MinSize:     20,
		MaxSize:     1000,
		ShiftFactor: 0.1,
		ScaleFactor: 1.1,

		ImageParams: pigo.ImageParams{
			Pixels: pixels,
			Rows:   rows,
			Cols:   cols,
			Dim:    cols,
		},
	}

	p := pigo.NewPigo()
	// Unpack the binary file. This will return the number of cascade trees,
	// the tree depth, the threshold and the prediction from tree's leaf nodes.
	classifier := p.Unpack()

	angle := 0.0 // cascade rotation angle. 0.0 is 0 radians and 1.0 is 2*pi radians

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	dets := classifier.RunCascade(cParams, angle)

	// Calculate the intersection over union (IoU) of two clusters.
	dets = classifier.ClusterDetections(dets, 0.2)

For pupil/eyes localization and facial landmark points detection API example check the source code.
*/
package pigo
