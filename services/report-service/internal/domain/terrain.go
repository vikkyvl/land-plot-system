package domain

import (
	"math"
	"slices"
)

const (
	meanEarthRadiusM   = 6_371_008.8
	degreesInHalfTurn  = 180
	minimumGridSide    = 2
	minimumFitSamples  = 3
	percentMultiplier  = 100
	planeDeterminantEp = 1e-9
)

type DataStatus string

const (
	DataAvailable   DataStatus = "available"
	DataUnavailable DataStatus = "unavailable"
)

type Terrain struct {
	Status                 DataStatus
	SlopeDegrees           float64
	SurfaceAreaM2          float64
	SurfaceIncreasePercent float64
	MinElevationM          float64
	MaxElevationM          float64
	ReliefM                float64
	Coarse                 bool
}

type TerrainSettings struct {
	GridSide     int
	MarginM      float64
	CoarseAreaM2 float64
}

type localPoint struct {
	x float64
	y float64
}

type localPlane struct {
	originLatitude  float64
	originLongitude float64
	metersPerRadian float64
	parallelScale   float64
}

const (
	TerrainSource = "Open-Meteo.com (CC BY 4.0), Copernicus DEM GLO-90"
	ClimateSource = "Open-Meteo.com (CC BY 4.0), Copernicus C3S ERA5"
)

func UnavailableTerrain() Terrain {
	return Terrain{Status: DataUnavailable}
}

func (settings TerrainSettings) Grid(boundary []Coordinate) []Coordinate {
	plane := newLocalPlane(boundary)
	minimum, maximum := plane.bounds(boundary)
	side := max(settings.GridSide, minimumGridSide)
	steps := float64(side - 1)

	grid := make([]Coordinate, 0, side*side)
	for row := range side {
		for column := range side {
			grid = append(grid, plane.coordinate(localPoint{
				x: minimum.x - settings.MarginM + (maximum.x-minimum.x+2*settings.MarginM)*float64(column)/steps,
				y: minimum.y - settings.MarginM + (maximum.y-minimum.y+2*settings.MarginM)*float64(row)/steps,
			}))
		}
	}

	return grid
}

func (settings TerrainSettings) Profile(grid []Coordinate, elevations []float64, areaM2 float64) (Terrain, error) {
	if len(grid) != len(elevations) || len(grid) < minimumFitSamples {
		return Terrain{}, ErrElevationCountMismatch
	}

	gradientX, gradientY, err := fitPlane(newLocalPlane(grid).project(grid), elevations)
	if err != nil {
		return Terrain{}, err
	}

	slope := math.Atan(math.Hypot(gradientX, gradientY))
	lowest, highest := slices.Min(elevations), slices.Max(elevations)

	return Terrain{
		Status:                 DataAvailable,
		SlopeDegrees:           radiansToDegrees(slope),
		SurfaceAreaM2:          areaM2 / math.Cos(slope),
		SurfaceIncreasePercent: (1/math.Cos(slope) - 1) * percentMultiplier,
		MinElevationM:          lowest,
		MaxElevationM:          highest,
		ReliefM:                highest - lowest,
		Coarse:                 areaM2 < settings.CoarseAreaM2,
	}, nil
}

func fitPlane(points []localPoint, heights []float64) (float64, float64, error) {
	count := float64(len(points))

	var meanX, meanY, meanZ float64
	for i, point := range points {
		meanX += point.x / count
		meanY += point.y / count
		meanZ += heights[i] / count
	}

	var sumXX, sumXY, sumYY, sumXZ, sumYZ float64

	for i, point := range points {
		deltaX, deltaY, deltaZ := point.x-meanX, point.y-meanY, heights[i]-meanZ
		sumXX += deltaX * deltaX
		sumXY += deltaX * deltaY
		sumYY += deltaY * deltaY
		sumXZ += deltaX * deltaZ
		sumYZ += deltaY * deltaZ
	}

	determinant := sumXX*sumYY - sumXY*sumXY
	if math.Abs(determinant) <= planeDeterminantEp*(sumXX*sumYY+1) {
		return 0, 0, ErrDegenerateTerrainGrid
	}

	return (sumXZ*sumYY - sumYZ*sumXY) / determinant, (sumYZ*sumXX - sumXZ*sumXY) / determinant, nil
}

func newLocalPlane(points []Coordinate) localPlane {
	var latitude, longitude float64
	for _, point := range points {
		latitude += point.Latitude / float64(len(points))
		longitude += point.Longitude / float64(len(points))
	}

	return localPlane{
		originLatitude:  latitude,
		originLongitude: longitude,
		metersPerRadian: meanEarthRadiusM,
		parallelScale:   math.Cos(degreesToRadians(latitude)),
	}
}

func (plane localPlane) point(coordinate Coordinate) localPoint {
	return localPoint{
		x: degreesToRadians(coordinate.Longitude-plane.originLongitude) * plane.metersPerRadian * plane.parallelScale,
		y: degreesToRadians(coordinate.Latitude-plane.originLatitude) * plane.metersPerRadian,
	}
}

func (plane localPlane) coordinate(point localPoint) Coordinate {
	return Coordinate{
		Latitude:  plane.originLatitude + radiansToDegrees(point.y/plane.metersPerRadian),
		Longitude: plane.originLongitude + radiansToDegrees(point.x/(plane.metersPerRadian*plane.parallelScale)),
	}
}

func (plane localPlane) project(coordinates []Coordinate) []localPoint {
	points := make([]localPoint, len(coordinates))
	for i, coordinate := range coordinates {
		points[i] = plane.point(coordinate)
	}

	return points
}

func (plane localPlane) bounds(coordinates []Coordinate) (localPoint, localPoint) {
	points := plane.project(coordinates)
	minimum, maximum := points[0], points[0]

	for _, point := range points[1:] {
		minimum = localPoint{x: math.Min(minimum.x, point.x), y: math.Min(minimum.y, point.y)}
		maximum = localPoint{x: math.Max(maximum.x, point.x), y: math.Max(maximum.y, point.y)}
	}

	return minimum, maximum
}

func degreesToRadians(value float64) float64 { return value * math.Pi / degreesInHalfTurn }

func radiansToDegrees(value float64) float64 { return value * degreesInHalfTurn / math.Pi }
