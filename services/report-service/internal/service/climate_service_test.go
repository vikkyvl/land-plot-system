package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

var (
	errWeatherDown      = errors.New("weather is down")
	errCacheDown        = errors.New("cache is down")
	testClimateNow      = time.Date(2026, time.November, 18, 12, 0, 0, 0, time.UTC)
	testClimateSettings = ClimateSettings{Years: 10, CacheTTL: 365 * 24 * time.Hour}
	testClimateCentre   = domain.Coordinate{Latitude: 50.4501, Longitude: 30.5234}
	testClimateCacheKey = ClimateCacheKey{
		Cell:   domain.ClimateCell{LatitudeTenths: 505, LongitudeTenths: 305},
		Period: domain.ClimatePeriod{FirstYear: 2016, LastYear: 2025},
	}
	testDailyWeatherDays = []domain.DailyWeather{
		{Date: time.Date(2020, time.July, 1, 0, 0, 0, 0, time.UTC), MeanC: 20, MinC: 15, MaxC: 25},
	}
)

type climateProviderStub struct {
	err   error
	point domain.Coordinate
	from  time.Time
	to    time.Time
	calls int
}

func (stub *climateProviderStub) DailyWeather(
	_ context.Context,
	point domain.Coordinate,
	from, to time.Time,
) ([]domain.DailyWeather, error) {
	stub.calls++
	stub.point, stub.from, stub.to = point, from, to

	return testDailyWeatherDays, stub.err
}

type climateCacheStub struct {
	stored    map[ClimateCacheKey]domain.Climate
	notBefore time.Time
	err       error
}

func (stub *climateCacheStub) Get(
	_ context.Context,
	key ClimateCacheKey,
	notBefore time.Time,
) (domain.Climate, bool, error) {
	stub.notBefore = notBefore
	climate, found := stub.stored[key]

	return climate, found, stub.err
}

func (stub *climateCacheStub) Put(_ context.Context, key ClimateCacheKey, climate domain.Climate, _ time.Time) error {
	if stub.err != nil {
		return stub.err
	}

	stub.stored[key] = climate

	return nil
}

func newTestClimateService(provider ClimateProvider, cache ClimateCache) *ClimateService {
	return NewClimateService(provider, cache, testClimateSettings, slog.New(slog.DiscardHandler))
}

func TestClimateIsFetchedForTheCellCentreAndCached(t *testing.T) {
	provider := &climateProviderStub{}
	cache := &climateCacheStub{stored: map[ClimateCacheKey]domain.Climate{}}
	climates := newTestClimateService(provider, cache)

	climate, err := climates.Profile(context.Background(), testClimateCentre, testClimateNow)
	if err != nil || climate.Status != domain.DataAvailable {
		t.Fatalf("Profile() = %+v, %v", climate, err)
	}

	if provider.point != (domain.Coordinate{Latitude: 50.5, Longitude: 30.5}) ||
		provider.from != time.Date(2015, time.December, 1, 0, 0, 0, 0, time.UTC) ||
		provider.to != time.Date(2025, time.December, 31, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("asked %+v from %v to %v", provider.point, provider.from, provider.to)
	}

	if _, found := cache.stored[testClimateCacheKey]; !found {
		t.Fatalf("cache = %+v", cache.stored)
	}

	if _, err := climates.Profile(
		context.Background(),
		testClimateCentre,
		testClimateNow,
	); err != nil ||
		provider.calls != 1 {
		t.Fatalf("second Profile() error = %v after %d calls", err, provider.calls)
	}

	if !cache.notBefore.Equal(testClimateNow.Add(-testClimateSettings.CacheTTL)) {
		t.Fatalf("cache asked for entries since %v", cache.notBefore)
	}
}

func TestClimateIsUnavailableWhenWeatherFails(t *testing.T) {
	cache := &climateCacheStub{stored: map[ClimateCacheKey]domain.Climate{}}

	climate, err := newTestClimateService(&climateProviderStub{err: errWeatherDown}, cache).
		Profile(context.Background(), testClimateCentre, testClimateNow)
	if !errors.Is(err, errWeatherDown) || climate.Status != domain.DataUnavailable || climate.FirstYear != 2016 ||
		len(cache.stored) != 0 {
		t.Fatalf("Profile() = %+v, %v", climate, err)
	}
}

func TestClimateWorksWhenTheCacheFails(t *testing.T) {
	provider := &climateProviderStub{}
	cache := &climateCacheStub{stored: map[ClimateCacheKey]domain.Climate{}, err: errCacheDown}

	climate, err := newTestClimateService(
		provider,
		cache,
	).Profile(context.Background(), testClimateCentre, testClimateNow)
	if err != nil || climate.Status != domain.DataAvailable || provider.calls != 1 {
		t.Fatalf("Profile() = %+v, %v", climate, err)
	}
}
