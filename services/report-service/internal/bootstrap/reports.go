package bootstrap

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/client/httpclient"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/client/openmeteo"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/config"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/discovery/consul"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/repository/postgres"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/service"
)

func newReportService(serviceConfig config.Config, pool *pgxpool.Pool, logger *slog.Logger) *service.ReportService {
	resolver := consul.NewResolver(serviceConfig.ConsulAddress, &http.Client{Timeout: serviceConfig.ConsulTimeout})
	plotClient := httpclient.NewPlotClient(
		resolver,
		&http.Client{Timeout: serviceConfig.UpstreamTimeout},
		serviceConfig.PlotServiceName,
	)

	return service.NewReportService(
		postgres.NewReportRepository(pool),
		plotClient,
		newSurroundings(serviceConfig, pool, logger),
		serviceConfig.InstanceID,
	)
}

func newSurroundings(
	serviceConfig config.Config,
	pool *pgxpool.Pool,
	logger *slog.Logger,
) *service.SurroundingsService {
	openMeteo := serviceConfig.OpenMeteo
	terrain := service.NewTerrainService(
		openmeteo.NewElevationClient(openMeteo.ElevationURL, openMeteo.Timeout),
		domain.TerrainSettings{
			GridSide:     serviceConfig.Terrain.GridSide,
			MarginM:      serviceConfig.Terrain.MarginM,
			CoarseAreaM2: serviceConfig.Terrain.CoarseAreaM2,
		},
	)
	climate := service.NewClimateService(
		openmeteo.NewArchiveClient(openMeteo.ArchiveURL, openMeteo.Timeout),
		postgres.NewClimateCacheRepository(pool),
		service.ClimateSettings{Years: serviceConfig.Climate.Years, CacheTTL: serviceConfig.Climate.CacheTTL},
		logger,
	)

	return service.NewSurroundingsService(terrain, climate, logger)
}
