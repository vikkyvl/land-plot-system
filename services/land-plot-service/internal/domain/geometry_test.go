package domain

import (
	"errors"
	"math"
	"testing"
)

func TestInspectPolygonReportsTheFirstProblem(t *testing.T) {
	tests := []struct {
		name   string
		points []Coordinate
		want   error
	}{
		{
			name:   "valid polygon",
			points: referencePolygon(),
			want:   nil,
		},
		{
			name:   "more points than allowed",
			points: make([]Coordinate, MaximumPolygonPoints+1),
			want:   ErrTooManyPoints,
		},
		{
			name:   "fewer than three points",
			points: []Coordinate{{Latitude: 50.45, Longitude: 30.52}, {Latitude: 50.45, Longitude: 30.5214}},
			want:   ErrInsufficientPoints,
		},
		{
			name: "latitude out of range",
			points: []Coordinate{
				{Latitude: 91, Longitude: 0},
				{Latitude: 0, Longitude: 1},
				{Latitude: 1, Longitude: 0},
			},
			want: ErrInvalidCoordinate,
		},
		{
			name: "longitude out of range",
			points: []Coordinate{
				{Latitude: 0, Longitude: 181},
				{Latitude: 0, Longitude: 1},
				{Latitude: 1, Longitude: 0},
			},
			want: ErrInvalidCoordinate,
		},
		{
			name: "not a number",
			points: []Coordinate{
				{Latitude: math.NaN(), Longitude: 0},
				{Latitude: 0, Longitude: 1},
				{Latitude: 1, Longitude: 0},
			},
			want: ErrInvalidCoordinate,
		},
		{
			name: "duplicate point",
			points: []Coordinate{
				{
					Latitude:  50.45,
					Longitude: 30.52,
				},
				{Latitude: 50.45, Longitude: 30.5214},
				{Latitude: 50.45, Longitude: 30.52},
			},
			want: ErrDuplicatePoint,
		},
		{
			name: "self-intersection",
			points: []Coordinate{
				{Latitude: 50.45, Longitude: 30.52},
				{Latitude: 50.4509, Longitude: 30.5214},
				{Latitude: 50.45, Longitude: 30.5214},
				{Latitude: 50.4509, Longitude: 30.52},
			},
			want: ErrSelfIntersection,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := measure(test.points, LocalEllipsoidCalculator{})
			if !errors.Is(err, test.want) {
				t.Fatalf("measure() error = %v, want %v", err, test.want)
			}
		})
	}
}

func hourglass() []Coordinate {
	return []Coordinate{
		{Latitude: 50.4509, Longitude: 30.52},
		{Latitude: 50.4509, Longitude: 30.5214},
		{Latitude: 50.45, Longitude: 30.52},
		{Latitude: 50.45, Longitude: 30.5214},
	}
}

func TestSelfIntersectionNamesTheCrossingSidesAndThePointsToSwap(t *testing.T) {
	inspection := InspectPolygon(hourglass(), LocalEllipsoidCalculator{})

	if inspection.Valid() || len(inspection.Errors) != 1 || !errors.Is(inspection.Errors[0], ErrSelfIntersection) {
		t.Fatalf("errors = %v", inspection.Errors)
	}

	details, ok := inspection.Errors[0].Details.(SelfIntersectionDetails)
	if !ok || details.Segments != [2][2]int{{1, 2}, {3, 0}} || details.Swap != [2]int{2, 3} {
		t.Fatalf("details = %#v", inspection.Errors[0].Details)
	}

	if math.Abs(details.Point.Latitude-50.45045) > 1e-9 || math.Abs(details.Point.Longitude-30.5207) > 1e-9 {
		t.Fatalf("crossing point = %+v, want the middle of the hourglass", details.Point)
	}
}

func TestCrossingPointOfSidesTouchingAtAVertex(t *testing.T) {
	first := segment{start: metricPoint{x: 0, y: 0}, end: metricPoint{x: 10, y: 0}}
	touching := segment{start: metricPoint{x: 5, y: 0}, end: metricPoint{x: 5, y: 7}}
	overlapping := segment{start: metricPoint{x: 3, y: 0}, end: metricPoint{x: 20, y: 0}}

	if point := crossingPoint(first, touching); point != (metricPoint{x: 5, y: 0}) {
		t.Fatalf("touching point = %+v", point)
	}

	if point := crossingPoint(first, overlapping); point != (metricPoint{x: 3, y: 0}) {
		t.Fatalf("overlapping point = %+v", point)
	}
}

func TestSwappingTheSuggestedPointsRemovesTheIntersection(t *testing.T) {
	points := hourglass()
	details := InspectPolygon(points, LocalEllipsoidCalculator{}).Errors[0].Details.(SelfIntersectionDetails)

	points[details.Swap[0]], points[details.Swap[1]] = points[details.Swap[1]], points[details.Swap[0]]

	if inspection := InspectPolygon(points, LocalEllipsoidCalculator{}); !inspection.Valid() {
		t.Fatalf("errors after the swap = %v", inspection.Errors)
	}
}

func TestInspectPolygonCollectsIndependentPointProblems(t *testing.T) {
	points := []Coordinate{{Latitude: 50, Longitude: 30}, {Latitude: 95, Longitude: 30}, {Latitude: 50, Longitude: 30}}

	inspection := InspectPolygon(points, LocalEllipsoidCalculator{})

	if len(inspection.Errors) != 2 {
		t.Fatalf("errors = %v", inspection.Errors)
	}

	coordinate, duplicate := inspection.Errors[0], inspection.Errors[1]
	if !errors.Is(coordinate, ErrInvalidCoordinate) || coordinate.Details != (InvalidCoordinateDetails{Index: 1}) {
		t.Fatalf("coordinate error = %#v", coordinate)
	}

	if !errors.Is(duplicate, ErrDuplicatePoint) || duplicate.Details != (DuplicatePointDetails{Indices: [2]int{0, 2}}) {
		t.Fatalf("duplicate error = %#v", duplicate)
	}
}

func TestInspectPolygonSkipsTheIntersectionCheckForTooFewPoints(t *testing.T) {
	inspection := InspectPolygon(referencePolygon()[:2], LocalEllipsoidCalculator{})

	if len(inspection.Errors) != 1 || !errors.Is(inspection.Errors[0], ErrInsufficientPoints) {
		t.Fatalf("errors = %v", inspection.Errors)
	}
}

func TestValidPolygonHasAreaAndPerimeter(t *testing.T) {
	inspection := InspectPolygon(referencePolygon(), LocalEllipsoidCalculator{})

	if !inspection.Valid() || inspection.Area.AreaM2 <= 0 || inspection.Area.PerimeterM <= 0 {
		t.Fatalf("inspection = %#v", inspection)
	}
}

func TestAreaUnits(t *testing.T) {
	if SquareMetersToAres(24_810) != 248.1 || SquareMetersToHectares(24_810) != 2.481 {
		t.Fatalf("ares = %v, hectares = %v", SquareMetersToAres(24_810), SquareMetersToHectares(24_810))
	}
}
