package domain

import (
	"math"
	"slices"
	"testing"
)

const (
	referenceAreaM2     = 9954.64
	referencePerimeterM = 399.09
	relativeTolerance   = 0.0001
)

func referencePolygon() []Coordinate {
	return []Coordinate{
		{Latitude: 50.45, Longitude: 30.52},
		{Latitude: 50.45, Longitude: 30.5214},
		{Latitude: 50.4509, Longitude: 30.5214},
		{Latitude: 50.4509, Longitude: 30.52},
	}
}

func reversed(points []Coordinate) []Coordinate {
	result := slices.Clone(points)
	slices.Reverse(result)

	return result
}

func shiftedStart(points []Coordinate) []Coordinate {
	return append(slices.Clone(points[1:]), points[0])
}

func withinTolerance(got, want float64) bool {
	return math.Abs(got-want) <= want*relativeTolerance
}

func TestLocalEllipsoidCalculatorMatchesGeographicLib(t *testing.T) {
	tests := []struct {
		name   string
		points []Coordinate
	}{
		{name: "reference order", points: referencePolygon()},
		{name: "reversed order", points: reversed(referencePolygon())},
		{name: "shifted start point", points: shiftedStart(referencePolygon())},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := LocalEllipsoidCalculator{}.Calculate(test.points)

			if !withinTolerance(result.AreaM2, referenceAreaM2) {
				t.Fatalf(
					"area = %.4f m², want %.2f m² ± %.2f %%",
					result.AreaM2,
					referenceAreaM2,
					relativeTolerance*100,
				)
			}

			if !withinTolerance(result.PerimeterM, referencePerimeterM) {
				t.Fatalf("perimeter = %.4f m, want %.2f m ± %.2f %%",
					result.PerimeterM, referencePerimeterM, relativeTolerance*100)
			}
		})
	}
}

func TestLocalEllipsoidCalculatorIgnoresOrderAndStart(t *testing.T) {
	calculator := LocalEllipsoidCalculator{}
	want := calculator.Calculate(referencePolygon())

	for _, points := range [][]Coordinate{reversed(referencePolygon()), shiftedStart(referencePolygon())} {
		got := calculator.Calculate(points)

		if math.Abs(got.AreaM2-want.AreaM2) > 1e-6 || math.Abs(got.PerimeterM-want.PerimeterM) > 1e-6 {
			t.Fatalf("Calculate(%v) = %#v, want %#v", points, got, want)
		}
	}
}
