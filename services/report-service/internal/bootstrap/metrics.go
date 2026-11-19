package bootstrap

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/config"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/metrics"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/repository/postgres"
)

func newMetrics(serviceConfig config.Config, pool *pgxpool.Pool) (*postgres.OutboxRepository, *metrics.Service) {
	outboxRepository := postgres.NewOutboxRepository(pool)

	return outboxRepository, metrics.New(config.ServiceName, metrics.Stats{
		Backlog: outboxRepository,
		Reports: postgres.NewReportRepository(pool),
		Timeout: serviceConfig.DatabaseTimeout,
	})
}
