package domain

import "math"

const (
	MinimumPolygonPoints = 3
	MaximumPolygonPoints = 1000
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

type PolygonInspection struct {
	Area   AreaResult
	Errors []*Error
}

func (inspection PolygonInspection) Valid() bool {
	return len(inspection.Errors) == 0
}

type metricPoint struct {
	x float64
	y float64
}

func InspectPolygon(points []Coordinate, calculator AreaCalculator) PolygonInspection {
	if len(points) > MaximumPolygonPoints {
		return PolygonInspection{Errors: []*Error{ErrTooManyPoints}}
	}

	if problems := pointProblems(points); len(problems) > 0 {
		return PolygonInspection{Errors: problems}
	}

	if intersection, found := findSelfIntersection(projectToLocalPlane(points)); found {
		return PolygonInspection{Errors: []*Error{ErrSelfIntersection.WithDetails(intersection)}}
	}

	area := calculator.Calculate(points)
	if area.AreaM2 <= minimumAreaM2 {
		return PolygonInspection{Errors: []*Error{ErrZeroArea}}
	}

	return PolygonInspection{Area: area}
}

func measure(points []Coordinate, calculator AreaCalculator) (AreaResult, error) {
	inspection := InspectPolygon(points, calculator)
	if !inspection.Valid() {
		return AreaResult{}, inspection.Errors[0]
	}

	return inspection.Area, nil
}

func pointProblems(points []Coordinate) []*Error {
	var problems []*Error

	if len(points) < MinimumPolygonPoints {
		problems = append(problems, ErrInsufficientPoints)
	}

	if index, found := firstInvalidCoordinate(points); found {
		problems = append(problems, ErrInvalidCoordinate.WithDetails(InvalidCoordinateDetails{Index: index}))
	}

	if first, second, found := firstDuplicatePair(points); found {
		problems = append(
			problems,
			ErrDuplicatePoint.WithDetails(DuplicatePointDetails{Indices: [2]int{first, second}}),
		)
	}

	return problems
}

func firstInvalidCoordinate(points []Coordinate) (int, bool) {
	for index, point := range points {
		if !point.isValid() {
			return index, true
		}
	}

	return 0, false
}

func firstDuplicatePair(points []Coordinate) (int, int, bool) {
	seen := make(map[Coordinate]int, len(points))
	for index, point := range points {
		if earlier, exists := seen[point]; exists {
			return earlier, index, true
		}

		seen[point] = index
	}

	return 0, 0, false
}

func degreesToRadians(value float64) float64 { return value * math.Pi / degreesInHalfTurn }
