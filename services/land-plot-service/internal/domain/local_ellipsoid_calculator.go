package domain

import "math"

const (
	WGS84SemiMajorAxis       = 6378137.0
	WGS84Flattening          = 1 / 298.257223563
	wgs84AxisRatio           = 1 - WGS84Flattening
	wgs84EccentricitySquared = 1 - wgs84AxisRatio*wgs84AxisRatio
)

const shoelaceAreaDivisor = 2

type LocalEllipsoidCalculator struct{}

func (LocalEllipsoidCalculator) Calculate(points []Coordinate) AreaResult {
	projected := projectToLocalPlane(points)

	var twiceSignedArea, perimeter float64

	for i, point := range projected {
		next := projected[(i+1)%len(projected)]
		twiceSignedArea += point.x*next.y - next.x*point.y
		perimeter += math.Hypot(next.x-point.x, next.y-point.y)
	}

	return AreaResult{
		AreaM2:     math.Abs(twiceSignedArea) / shoelaceAreaDivisor,
		PerimeterM: perimeter,
	}
}

func projectToLocalPlane(points []Coordinate) []metricPoint {
	var latitudeCenter, longitudeCenter float64
	for _, point := range points {
		latitudeCenter += point.Latitude
		longitudeCenter += point.Longitude
	}

	latitudeCenter /= float64(len(points))
	longitudeCenter /= float64(len(points))

	centerLatitude := degreesToRadians(latitudeCenter)
	sinLatitude := math.Sin(centerLatitude)
	curvature := 1 - wgs84EccentricitySquared*sinLatitude*sinLatitude
	meridianRadius := WGS84SemiMajorAxis * (1 - wgs84EccentricitySquared) / (curvature * math.Sqrt(curvature))
	primeVerticalRadius := WGS84SemiMajorAxis / math.Sqrt(curvature)
	parallelRadius := primeVerticalRadius * math.Cos(centerLatitude)

	projected := make([]metricPoint, len(points))
	for i, point := range points {
		projected[i] = metricPoint{
			x: degreesToRadians(point.Longitude-longitudeCenter) * parallelRadius,
			y: degreesToRadians(point.Latitude-latitudeCenter) * meridianRadius,
		}
	}

	return projected
}
