package bootstrap

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/config"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/faults"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/metrics"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/repository/postgres"
)

func newMetrics(
	serviceConfig config.Config,
	pool *pgxpool.Pool,
	plotRepository *postgres.PlotRepository,
	logger *slog.Logger,
) (*postgres.OutboxRepository, *metrics.Service, *faults.Switch) {
	outboxRepository := postgres.NewOutboxRepository(pool)
	serviceMetrics := metrics.New(config.ServiceName, metrics.Stats{
		Backlog: outboxRepository,
		Plots:   plotRepository,
		Timeout: serviceConfig.DatabaseTimeout,
	})
	faultSwitch := faults.NewSwitch(serviceConfig.DemoFaultsEnabled, logger)
	serviceMetrics.ObserveHealth(faultSwitch.Healthy)

	return outboxRepository, serviceMetrics, faultSwitch
}
