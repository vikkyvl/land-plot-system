package domain

import "math"

type OutlinePoint struct {
	X float64
	Y float64
}

func Outline(points []Coordinate) []OutlinePoint {
	if len(points) == 0 {
		return []OutlinePoint{}
	}

	projected := projectToLocalPlane(points)
	minimum, maximum := projected[0], projected[0]

	for _, point := range projected[1:] {
		minimum = metricPoint{x: math.Min(minimum.x, point.x), y: math.Min(minimum.y, point.y)}
		maximum = metricPoint{x: math.Max(maximum.x, point.x), y: math.Max(maximum.y, point.y)}
	}

	side := math.Max(maximum.x-minimum.x, maximum.y-minimum.y)
	if side <= 0 {
		side = 1
	}

	offsetX := (side - (maximum.x - minimum.x)) / outlineHalves
	offsetY := (side - (maximum.y - minimum.y)) / outlineHalves

	outline := make([]OutlinePoint, len(projected))
	for index, point := range projected {
		outline[index] = OutlinePoint{
			X: (point.x - minimum.x + offsetX) / side,
			Y: (maximum.y - point.y + offsetY) / side,
		}
	}

	return outline
}

const outlineHalves = 2
