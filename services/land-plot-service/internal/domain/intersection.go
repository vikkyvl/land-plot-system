package domain

import "math"

const geometryEpsilon = 1e-9

type segment struct {
	start metricPoint
	end   metricPoint
}

func (edge segment) orientationOf(point metricPoint) float64 {
	return (edge.end.x-edge.start.x)*(point.y-edge.start.y) - (edge.end.y-edge.start.y)*(point.x-edge.start.x)
}

func (edge segment) contains(point metricPoint) bool {
	return point.x >= math.Min(edge.start.x, edge.end.x)-geometryEpsilon &&
		point.x <= math.Max(edge.start.x, edge.end.x)+geometryEpsilon &&
		point.y >= math.Min(edge.start.y, edge.end.y)-geometryEpsilon &&
		point.y <= math.Max(edge.start.y, edge.end.y)+geometryEpsilon
}

func selfIntersects(points []metricPoint) bool {
	count := len(points)
	for i := range points {
		first := segment{start: points[i], end: points[(i+1)%count]}
		for j := i + 1; j < count; j++ {
			if adjacentEdges(i, j, count) {
				continue
			}

			second := segment{start: points[j], end: points[(j+1)%count]}
			if segmentsIntersect(first, second) {
				return true
			}
		}
	}

	return false
}

func adjacentEdges(first, second, count int) bool {
	return first == second || (first+1)%count == second || (second+1)%count == first
}

func segmentsIntersect(first, second segment) bool {
	secondStartSide := first.orientationOf(second.start)
	secondEndSide := first.orientationOf(second.end)
	firstStartSide := second.orientationOf(first.start)
	firstEndSide := second.orientationOf(first.end)

	if oppositeSigns(secondStartSide, secondEndSide) && oppositeSigns(firstStartSide, firstEndSide) {
		return true
	}

	return nearZero(secondStartSide) && first.contains(second.start) ||
		nearZero(secondEndSide) && first.contains(second.end) ||
		nearZero(firstStartSide) && second.contains(first.start) ||
		nearZero(firstEndSide) && second.contains(first.end)
}

func oppositeSigns(first, second float64) bool {
	return first > geometryEpsilon && second < -geometryEpsilon ||
		first < -geometryEpsilon && second > geometryEpsilon
}

func nearZero(value float64) bool { return math.Abs(value) <= geometryEpsilon }
