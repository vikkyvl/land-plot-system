package service

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

const barrierTimeout = time.Second

type barrier struct {
	mutex    *sync.Mutex
	waiting  *int
	released chan struct{}
}

func newBarrier(parties int) barrier {
	return barrier{mutex: &sync.Mutex{}, waiting: &parties, released: make(chan struct{})}
}

func (gate barrier) meet() bool {
	gate.mutex.Lock()
	*gate.waiting--

	if *gate.waiting == 0 {
		close(gate.released)
	}
	gate.mutex.Unlock()

	select {
	case <-gate.released:
		return true
	case <-time.After(barrierTimeout):
		return false
	}
}

type barrierElevations struct {
	gate barrier
	err  error
}

func (stub *barrierElevations) Elevations(_ context.Context, points []domain.Coordinate) ([]float64, error) {
	if !stub.gate.meet() {
		return nil, errElevationsDown
	}

	if stub.err != nil {
		return nil, stub.err
	}

	return make([]float64, len(points)), nil
}

type barrierWeather struct {
	gate barrier
}

func (stub *barrierWeather) DailyWeather(
	context.Context,
	domain.Coordinate,
	time.Time,
	time.Time,
) ([]domain.DailyWeather, error) {
	if !stub.gate.meet() {
		return nil, errWeatherDown
	}

	return testDailyWeatherDays, nil
}

func describeWithBarrier(t *testing.T, elevationErr error) (domain.Terrain, domain.Climate) {
	t.Helper()

	gate := newBarrier(2)
	logger := slog.New(slog.DiscardHandler)
	surroundings := NewSurroundingsService(
		NewTerrainService(&barrierElevations{gate: gate, err: elevationErr}, testTerrainSettings),
		NewClimateService(&barrierWeather{gate: gate}, &climateCacheStub{stored: map[ClimateCacheKey]domain.Climate{}},
			testClimateSettings, logger),
		logger,
	)
	surroundings.now = func() time.Time { return testClimateNow }

	return surroundings.Describe(context.Background(), domain.PlotSnapshot{Points: testTerrainBoundary, AreaM2: 20_000})
}

func TestSurroundingsAskBothSourcesAtOnce(t *testing.T) {
	terrain, climate := describeWithBarrier(t, nil)

	if terrain.Status != domain.DataAvailable || climate.Status != domain.DataAvailable {
		t.Fatalf("terrain = %+v, climate = %+v", terrain, climate)
	}
}

func TestSurroundingsKeepWhatArrived(t *testing.T) {
	terrain, climate := describeWithBarrier(t, errElevationsDown)

	if terrain.Status != domain.DataUnavailable || climate.Status != domain.DataAvailable || climate.LastYear != 2025 {
		t.Fatalf("terrain = %+v, climate = %+v", terrain, climate)
	}
}
