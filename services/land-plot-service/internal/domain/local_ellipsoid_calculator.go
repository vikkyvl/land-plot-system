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

type localPlane struct {
	latitudeCenter  float64
	longitudeCenter float64
	meridianRadius  float64
	parallelRadius  float64
}

func newLocalPlane(points []Coordinate) localPlane {
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
	primeVerticalRadius := WGS84SemiMajorAxis / math.Sqrt(curvature)

	return localPlane{
		latitudeCenter:  latitudeCenter,
		longitudeCenter: longitudeCenter,
		meridianRadius:  WGS84SemiMajorAxis * (1 - wgs84EccentricitySquared) / (curvature * math.Sqrt(curvature)),
		parallelRadius:  primeVerticalRadius * math.Cos(centerLatitude),
	}
}

func (plane localPlane) project(points []Coordinate) []metricPoint {
	projected := make([]metricPoint, len(points))
	for i, point := range points {
		projected[i] = metricPoint{
			x: degreesToRadians(point.Longitude-plane.longitudeCenter) * plane.parallelRadius,
			y: degreesToRadians(point.Latitude-plane.latitudeCenter) * plane.meridianRadius,
		}
	}

	return projected
}

func (plane localPlane) coordinate(point metricPoint) Coordinate {
	return Coordinate{
		Latitude:  plane.latitudeCenter + radiansToDegrees(point.y/plane.meridianRadius),
		Longitude: plane.longitudeCenter + radiansToDegrees(point.x/plane.parallelRadius),
	}
}

func projectToLocalPlane(points []Coordinate) []metricPoint {
	return newLocalPlane(points).project(points)
}
