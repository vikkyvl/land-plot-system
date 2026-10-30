package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var testNow = time.Date(2026, time.October, 26, 12, 0, 0, 0, time.UTC)

var testCalculator = LocalEllipsoidCalculator{}

func TestNewPlotCalculatesPolygonAndStartsAtVersionOne(t *testing.T) {
	plot, err := NewPlot(
		"plot-1",
		PlotDraft{OwnerID: "owner-1", Name: "  Plot  ", Points: referencePolygon()},
		testCalculator,
		testNow,
	)
	if err != nil {
		t.Fatalf("NewPlot() error = %v", err)
	}

	if plot.Name != "Plot" || plot.Version != 1 || !plot.CreatedAt.Equal(plot.UpdatedAt) {
		t.Fatalf("NewPlot() plot = %#v", plot)
	}

	if !withinTolerance(plot.AreaM2, referenceAreaM2) || !withinTolerance(plot.PerimeterM, referencePerimeterM) {
		t.Fatalf("NewPlot() area = %f perimeter = %f", plot.AreaM2, plot.PerimeterM)
	}
}

func TestNewPlotRejectsInvalidName(t *testing.T) {
	for _, name := range []string{"   ", strings.Repeat("x", MaximumNameLength+1)} {
		_, err := NewPlot(
			"plot-1",
			PlotDraft{OwnerID: "owner-1", Name: name, Points: referencePolygon()},
			testCalculator,
			testNow,
		)
		if !errors.Is(err, ErrInvalidName) {
			t.Fatalf("NewPlot(%q) error = %v, want ErrInvalidName", name, err)
		}
	}
}

func TestNewPlotRejectsInvalidPolygon(t *testing.T) {
	tests := map[string]struct {
		points []Coordinate
		want   error
	}{
		"fewer than three points": {referencePolygon()[:2], ErrInsufficientPoints},
		"zero area": {
			[]Coordinate{
				{Latitude: 50.45, Longitude: 30.52},
				{Latitude: 50.45, Longitude: 30.5207},
				{Latitude: 50.45, Longitude: 30.5214},
			},
			ErrZeroArea,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := NewPlot(
				"plot-1",
				PlotDraft{OwnerID: "owner-1", Name: "Plot", Points: test.points},
				testCalculator,
				testNow,
			)
			if !errors.Is(err, test.want) {
				t.Fatalf("NewPlot() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestPlotUpdatedRecalculatesAndIncrementsVersion(t *testing.T) {
	plot, err := NewPlot(
		"plot-1",
		PlotDraft{OwnerID: "owner-1", Name: " Original ", Points: referencePolygon()},
		testCalculator,
		testNow,
	)
	if err != nil {
		t.Fatalf("NewPlot() error = %v", err)
	}

	name := " Updated "
	triangle := referencePolygon()[:3]

	updated, err := plot.Updated(PlotChanges{Name: &name, Points: &triangle}, testCalculator, testNow.Add(time.Hour))
	if err != nil {
		t.Fatalf("Updated() error = %v", err)
	}

	if updated.Name != "Updated" || updated.Version != 2 {
		t.Fatalf("Updated() plot = %#v", updated)
	}

	want := testCalculator.Calculate(triangle)
	if updated.AreaM2 != want.AreaM2 || updated.PerimeterM != want.PerimeterM {
		t.Fatalf("updated area = %f perimeter = %f, want %#v", updated.AreaM2, updated.PerimeterM, want)
	}
}

func TestPlotUpdatedRequiresChanges(t *testing.T) {
	plot, err := NewPlot(
		"plot-1",
		PlotDraft{OwnerID: "owner-1", Name: "Plot", Points: referencePolygon()},
		testCalculator,
		testNow,
	)
	if err != nil {
		t.Fatalf("NewPlot() error = %v", err)
	}

	_, err = plot.Updated(PlotChanges{}, testCalculator, testNow)
	if !errors.Is(err, ErrNoChanges) {
		t.Fatalf("Updated() error = %v, want ErrNoChanges", err)
	}
}
