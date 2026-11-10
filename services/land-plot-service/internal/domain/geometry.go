package domain

import "math"

const (
	MinimumPolygonPoints = 3
	minimumAreaM2        = 1e-6
	degreesInHalfTurn    = 180
)

type AreaResult struct {
	AreaM2     float64
	PerimeterM float64
}

type AreaCalculator interface {
	Calculate(points []Coordinate) AreaResult
}

type metricPoint struct {
	x float64
	y float64
}

func ValidatePolygon(points []Coordinate) error {
	if err := validatePoints(points); err != nil {
		return err
	}

	if selfIntersects(projectToLocalPlane(points)) {
		return ErrSelfIntersection
	}

	return nil
}

func measure(points []Coordinate, calculator AreaCalculator) (AreaResult, error) {
	if err := ValidatePolygon(points); err != nil {
		return AreaResult{}, err
	}

	result := calculator.Calculate(points)
	if result.AreaM2 <= minimumAreaM2 {
		return AreaResult{}, ErrZeroArea
	}

	return result, nil
}

func validatePoints(points []Coordinate) error {
	if len(points) < MinimumPolygonPoints {
		return ErrInsufficientPoints
	}

	seen := make(map[Coordinate]struct{}, len(points))
	for _, point := range points {
		if !point.isValid() {
			return ErrInvalidCoordinate
		}

		if _, exists := seen[point]; exists {
			return ErrDuplicatePoint
		}

		seen[point] = struct{}{}
	}

	return nil
}

func degreesToRadians(value float64) float64 { return value * math.Pi / degreesInHalfTurn }
