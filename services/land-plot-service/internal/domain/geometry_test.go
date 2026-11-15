package domain

import (
	"errors"
	"math"
	"testing"
)

func TestValidatePolygon(t *testing.T) {
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
			if err := ValidatePolygon(test.points); !errors.Is(err, test.want) {
				t.Fatalf("ValidatePolygon() error = %v, want %v", err, test.want)
			}
		})
	}
}
