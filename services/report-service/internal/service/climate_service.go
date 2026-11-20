package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/logging"
)

type ClimateSettings struct {
	Years    int
	CacheTTL time.Duration
}

type ClimateService struct {
	weather  ClimateProvider
	cache    ClimateCache
	settings ClimateSettings
	logger   *slog.Logger
}

func NewClimateService(
	weather ClimateProvider,
	cache ClimateCache,
	settings ClimateSettings,
	logger *slog.Logger,
) *ClimateService {
	return &ClimateService{weather: weather, cache: cache, settings: settings, logger: logger}
}

func (climateService *ClimateService) Profile(
	requestContext context.Context,
	centre domain.Coordinate,
	now time.Time,
) (domain.Climate, error) {
	cell := domain.CellOf(centre)
	key := ClimateCacheKey{Cell: cell, Period: domain.CompletedYears(now, climateService.settings.Years)}

	cached, found, err := climateService.cache.Get(requestContext, key, now.Add(-climateService.settings.CacheTTL))
	if err != nil {
		climateService.logger.WarnContext(requestContext, logReadClimateCache, logging.KeyError, err)
	}

	if found {
		return cached, nil
	}

	days, err := climateService.weather.DailyWeather(
		requestContext,
		cell.Centre(),
		key.Period.Start(),
		key.Period.End(),
	)
	if err != nil {
		return domain.UnavailableClimate(key.Period), fmt.Errorf(wrapGetDailyWeather, err)
	}

	climate := domain.ProfileClimate(days, key.Period)
	if climate.Status != domain.DataAvailable {
		return climate, nil
	}

	if err := climateService.cache.Put(requestContext, key, climate, now); err != nil {
		climateService.logger.WarnContext(requestContext, logWriteClimateCache, logging.KeyError, err)
	}

	return climate, nil
}
