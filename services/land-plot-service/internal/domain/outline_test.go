package domain

import (
	"math"
	"testing"
)

func TestOutlineFitsAUnitSquareWithNorthUp(t *testing.T) {
	outline := Outline(referencePolygon())

	for _, point := range outline {
		if point.X < -1e-9 || point.X > 1+1e-9 || point.Y < -1e-9 || point.Y > 1+1e-9 {
			t.Fatalf("outline point %+v is outside the unit square", point)
		}
	}

	wide := Outline([]Coordinate{
		{Latitude: 50, Longitude: 30}, {Latitude: 50, Longitude: 30.01}, {Latitude: 50.001, Longitude: 30.01},
	})
	if math.Abs(wide[0].X) > 1e-9 || math.Abs(wide[1].X-1) > 1e-9 || wide[2].Y >= wide[1].Y {
		t.Fatalf("outline = %+v, want the widest side to span the square and north on top", wide)
	}
}
