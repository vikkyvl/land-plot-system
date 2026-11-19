package bootstrap

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/config"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/metrics"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/repository/postgres"
)

func newMetrics(
	serviceConfig config.Config,
	pool *pgxpool.Pool,
	plotRepository *postgres.PlotRepository,
) (*postgres.OutboxRepository, *metrics.Service) {
	outboxRepository := postgres.NewOutboxRepository(pool)

	return outboxRepository, metrics.New(config.ServiceName, metrics.Stats{
		Backlog: outboxRepository,
		Plots:   plotRepository,
		Timeout: serviceConfig.DatabaseTimeout,
	})
}
