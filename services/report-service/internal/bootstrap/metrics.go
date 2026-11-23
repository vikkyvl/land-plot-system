package bootstrap

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/config"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/faults"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/metrics"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/repository/postgres"
)

func newMetrics(
	serviceConfig config.Config,
	pool *pgxpool.Pool,
	logger *slog.Logger,
) (*postgres.OutboxRepository, *metrics.Service, *faults.Switch) {
	outboxRepository := postgres.NewOutboxRepository(pool)
	serviceMetrics := metrics.New(config.ServiceName, metrics.Stats{
		Backlog: outboxRepository,
		Reports: postgres.NewReportRepository(pool),
		Timeout: serviceConfig.DatabaseTimeout,
	})
	faultSwitch := faults.NewSwitch(serviceConfig.DemoFaultsEnabled, logger)
	serviceMetrics.ObserveHealth(faultSwitch.Healthy)

	return outboxRepository, serviceMetrics, faultSwitch
}
