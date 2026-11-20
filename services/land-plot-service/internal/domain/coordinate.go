package domain

import "math"

const (
	minimumLatitude  = -90
	maximumLatitude  = 90
	minimumLongitude = -180
	maximumLongitude = 180
)

type Coordinate struct {
	Latitude  float64
	Longitude float64
}

func (coordinate Coordinate) isValid() bool {
	return finite(coordinate.Latitude) && finite(coordinate.Longitude) &&
		coordinate.Latitude >= minimumLatitude && coordinate.Latitude <= maximumLatitude &&
		coordinate.Longitude >= minimumLongitude && coordinate.Longitude <= maximumLongitude
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
