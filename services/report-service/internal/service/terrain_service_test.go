package service

import (
	"context"
	"errors"
	"testing"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

var (
	errElevationsDown   = errors.New("elevations are down")
	testTerrainSettings = domain.TerrainSettings{GridSide: 10, MarginM: 90, CoarseAreaM2: 10_000}
	testTerrainBoundary = []domain.Coordinate{
		{
			Latitude:  50.45,
			Longitude: 30.52,
		},
		{Latitude: 50.45, Longitude: 30.523},
		{Latitude: 50.452, Longitude: 30.523},
	}
)

type elevationProviderStub struct {
	height float64
	err    error
	asked  int
}

func (stub *elevationProviderStub) Elevations(_ context.Context, points []domain.Coordinate) ([]float64, error) {
	stub.asked = len(points)
	if stub.err != nil {
		return nil, stub.err
	}

	heights := make([]float64, len(points))
	for i := range heights {
		heights[i] = stub.height
	}

	return heights, nil
}

func TestTerrainProfileAsksForTheGrid(t *testing.T) {
	provider := &elevationProviderStub{height: 120}

	terrain, err := NewTerrainService(
		provider,
		testTerrainSettings,
	).Profile(context.Background(), testTerrainBoundary, 20_000)
	if err != nil || terrain.Status != domain.DataAvailable || terrain.MinElevationM != 120 || provider.asked != 100 {
		t.Fatalf("Profile() = %+v, %v after %d points", terrain, err, provider.asked)
	}
}

func TestTerrainIsUnavailableWhenElevationsFail(t *testing.T) {
	provider := &elevationProviderStub{err: errElevationsDown}

	terrain, err := NewTerrainService(
		provider,
		testTerrainSettings,
	).Profile(context.Background(), testTerrainBoundary, 20_000)
	if !errors.Is(err, errElevationsDown) || terrain.Status != domain.DataUnavailable {
		t.Fatalf("Profile() = %+v, %v", terrain, err)
	}
}
