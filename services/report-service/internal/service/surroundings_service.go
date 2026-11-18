package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/logging"
)

type SurroundingsService struct {
	terrain *TerrainService
	climate *ClimateService
	logger  *slog.Logger
	now     func() time.Time
}

func NewSurroundingsService(
	terrain *TerrainService,
	climate *ClimateService,
	logger *slog.Logger,
) *SurroundingsService {
	return &SurroundingsService{terrain: terrain, climate: climate, logger: logger, now: time.Now}
}

func (surroundings *SurroundingsService) Describe(
	requestContext context.Context,
	plot domain.PlotSnapshot,
) (domain.Terrain, domain.Climate) {
	var (
		terrain domain.Terrain
		climate domain.Climate
		group   sync.WaitGroup
	)

	group.Go(func() {
		var err error

		terrain, err = surroundings.terrain.Profile(requestContext, plot.Points, plot.AreaM2)
		if err != nil {
			surroundings.logger.WarnContext(requestContext, logTerrainUnavailable, logging.KeyError, err)
		}
	})

	group.Go(func() {
		var err error

		climate, err = surroundings.climate.Profile(requestContext, domain.Centroid(plot.Points), surroundings.now())
		if err != nil {
			surroundings.logger.WarnContext(requestContext, logClimateUnavailable, logging.KeyError, err)
		}
	})

	group.Wait()

	return terrain, climate
}
