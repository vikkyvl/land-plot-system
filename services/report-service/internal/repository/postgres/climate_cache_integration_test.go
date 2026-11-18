//go:build integration

package postgres

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/service"
)

func TestClimateCacheKeepsProfilesPerCellAndPeriod(t *testing.T) {
	repository := NewClimateCacheRepository(openTestPool(t))
	fetchedAt := time.Date(2026, time.November, 18, 12, 0, 0, 0, time.UTC)
	key := service.ClimateCacheKey{
		Cell:   domain.ClimateCell{LatitudeTenths: 505, LongitudeTenths: 305},
		Period: domain.ClimatePeriod{FirstYear: 2016, LastYear: 2025},
	}
	climate := domain.Climate{
		Status: domain.DataAvailable, FirstYear: 2016, LastYear: 2025, GrowingDays: 214.5,
		Seasons:          []domain.SeasonClimate{{Season: domain.SeasonWinter, MeanC: -2.1, FrostDays: 61.4}},
		LastSpringFrost:  &domain.MonthDay{Month: time.April, Day: 12},
		FirstAutumnFrost: nil,
	}

	if err := repository.Put(context.Background(), key, climate, fetchedAt); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	stored, found, err := repository.Get(context.Background(), key, fetchedAt.Add(-time.Hour))
	if err != nil || !found || !reflect.DeepEqual(stored, climate) {
		t.Fatalf("Get() = %+v, %v, %v", stored, found, err)
	}

	if _, found, err := repository.Get(context.Background(), key, fetchedAt.Add(time.Hour)); err != nil || found {
		t.Fatalf("expired Get() found = %v, error = %v", found, err)
	}

	other := key
	other.Period.LastYear = 2026

	if _, found, err := repository.Get(context.Background(), other, fetchedAt.Add(-time.Hour)); err != nil || found {
		t.Fatalf("other period found = %v, error = %v", found, err)
	}

	climate.GrowingDays = 220
	if err := repository.Put(context.Background(), key, climate, fetchedAt.Add(time.Minute)); err != nil {
		t.Fatalf("second Put() error = %v", err)
	}

	stored, _, _ = repository.Get(context.Background(), key, fetchedAt)
	if stored.GrowingDays != 220 {
		t.Fatalf("refreshed profile = %+v", stored)
	}
}
