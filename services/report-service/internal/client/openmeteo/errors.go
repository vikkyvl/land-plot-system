package openmeteo

import "errors"

var (
	ErrTooManyElevationPoints = errors.New("open-meteo elevation accepts at most 100 points per request")
	ErrUnexpectedResponse     = errors.New("open-meteo returned an unexpected response")
)
