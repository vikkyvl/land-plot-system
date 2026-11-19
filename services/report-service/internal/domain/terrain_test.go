package domain

import (
	"errors"
	"math"
	"testing"
)

const (
	testSlopeDegrees = 10.0
	testAreaM2       = 50_000.0
	testBaseHeightM  = 300.0
)

var (
	testTerrainSettings = TerrainSettings{GridSide: 10, MarginM: 90, CoarseAreaM2: 10_000}
	testBoundary        = []Coordinate{
		{Latitude: 48.16, Longitude: 24.50},
		{Latitude: 48.16, Longitude: 24.503},
		{Latitude: 48.162, Longitude: 24.503},
		{Latitude: 48.162, Longitude: 24.50},
	}
)

func tiltedElevations(grid []Coordinate, slopeDegrees, azimuthDegrees float64) []float64 {
	plane := newLocalPlane(grid)
	gradient := math.Tan(degreesToRadians(slopeDegrees))
	azimuth := degreesToRadians(azimuthDegrees)

	heights := make([]float64, len(grid))
	for i, point := range plane.project(grid) {
		heights[i] = testBaseHeightM + gradient*(point.x*math.Sin(azimuth)+point.y*math.Cos(azimuth))
	}

	return heights
}

func TestGridCoversBoundaryWithMargin(t *testing.T) {
	grid := testTerrainSettings.Grid(testBoundary)
	if len(grid) != 100 {
		t.Fatalf("grid has %d points, want 100", len(grid))
	}

	plane := newLocalPlane(testBoundary)
	boundaryMinimum, boundaryMaximum := plane.bounds(testBoundary)
	gridMinimum, gridMaximum := plane.bounds(grid)

	gaps := []float64{
		boundaryMinimum.x - gridMinimum.x, boundaryMinimum.y - gridMinimum.y,
		gridMaximum.x - boundaryMaximum.x, gridMaximum.y - boundaryMaximum.y,
	}
	for _, gap := range gaps {
		if math.Abs(gap-testTerrainSettings.MarginM) > 0.5 {
			t.Fatalf("margins %v m, want %v m", gaps, testTerrainSettings.MarginM)
		}
	}
}

func TestTenDegreeSlopeAddsOneAndAHalfPercent(t *testing.T) {
	grid := testTerrainSettings.Grid(testBoundary)

	for _, azimuth := range []float64{0, 45, 200} {
		terrain, err := testTerrainSettings.Profile(grid, tiltedElevations(grid, testSlopeDegrees, azimuth), testAreaM2)
		if err != nil {
			t.Fatalf("Profile() error = %v", err)
		}

		wantIncrease := (1/math.Cos(degreesToRadians(testSlopeDegrees)) - 1) * 100
		if math.Abs(terrain.SlopeDegrees-testSlopeDegrees) > 0.01 ||
			math.Abs(terrain.SurfaceIncreasePercent-wantIncrease) > 0.01 ||
			math.Abs(terrain.SurfaceIncreasePercent-1.54) > 0.01 ||
			math.Abs(terrain.SurfaceAreaM2-testAreaM2*(1+wantIncrease/100)) > 1 {
			t.Fatalf("azimuth %v: terrain = %+v", azimuth, terrain)
		}

		if terrain.Status != DataAvailable || terrain.Coarse || terrain.ReliefM <= 0 ||
			terrain.ReliefM != terrain.MaxElevationM-terrain.MinElevationM {
			t.Fatalf("azimuth %v: terrain = %+v", azimuth, terrain)
		}
	}
}

func TestFlatSmallPlotIsCoarseWithoutSlope(t *testing.T) {
	grid := testTerrainSettings.Grid(testBoundary)
	heights := make([]float64, len(grid))

	for i := range heights {
		heights[i] = testBaseHeightM
	}

	terrain, err := testTerrainSettings.Profile(grid, heights, 5_000)
	if err != nil || terrain.SlopeDegrees != 0 || terrain.SurfaceAreaM2 != 5_000 || !terrain.Coarse ||
		terrain.ReliefM != 0 || terrain.MinElevationM != testBaseHeightM {
		t.Fatalf("Profile() = %+v, %v", terrain, err)
	}
}

func TestProfileRejectsMismatchedElevations(t *testing.T) {
	grid := testTerrainSettings.Grid(testBoundary)

	if _, err := testTerrainSettings.Profile(
		grid,
		[]float64{1, 2},
		testAreaM2,
	); !errors.Is(
		err,
		ErrElevationCountMismatch,
	) {
		t.Fatalf("Profile() error = %v, want ErrElevationCountMismatch", err)
	}

	line := []Coordinate{
		{Latitude: 48, Longitude: 24},
		{Latitude: 48, Longitude: 24.001},
		{Latitude: 48, Longitude: 24.002},
	}
	if _, err := testTerrainSettings.Profile(
		line,
		[]float64{1, 2, 3},
		testAreaM2,
	); !errors.Is(
		err,
		ErrDegenerateTerrainGrid,
	) {
		t.Fatalf("Profile() error = %v, want ErrDegenerateTerrainGrid", err)
	}
}

func TestUnavailableTerrain(t *testing.T) {
	if UnavailableTerrain().Status != DataUnavailable {
		t.Fatal("unavailable terrain is not marked unavailable")
	}
}
