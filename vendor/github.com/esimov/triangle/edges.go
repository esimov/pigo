package triangle

import (
	"image"
	"math/rand"
	"time"
)

// PointRate defines the default point rate.
// Changing this value will modify the triangles sizes.
const PointRate = 0.875

// GetEdgePoints retrieves the triangle points after the Sobel threshold has been applied.
func GetEdgePoints(img *image.NRGBA, threshold, maxPoints int) []Point {
	rand.Seed(time.Now().UTC().UnixNano())
	width, height := img.Bounds().Max.X, img.Bounds().Max.Y

	var (
		sum, total     int
		x, y, sx, sy   int
		row, col, step int
		points         []Point
		dpoints        []Point
	)

	for y = 0; y < height; y++ {
		for x = 0; x < width; x++ {
			sum, total = 0, 0

			for row = -1; row <= 1; row++ {
				sy = y + row
				step = sy * width

				if sy >= 0 && sy < height {
					for col = -1; col <= 1; col++ {
						sx = x + col
						if sx >= 0 && sx < width {
							sum += int(img.Pix[(sx+step)*4])
							total++
						}
					}
				}
			}
			if total > 0 {
				sum /= total
			}
			if sum > threshold {
				points = append(points, Point{x: x, y: y})
			}
		}
	}
	ilen := len(points)
	tlen := ilen
	limit := int(float64(ilen) * PointRate)

	if limit > maxPoints {
		limit = maxPoints
	}

	for i := 0; i < limit && i < ilen; i++ {
		j := int(float64(tlen) * rand.Float64())
		dpoints = append(dpoints, points[j])
		// Remove points
		points = append(points[:j], points[j+1:]...)
		tlen--
	}
	return dpoints
}
