package bootstrap

import (
	"log/slog"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/config"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/service"
)

func newOutboxRelay(
	serviceConfig config.Config,
	repository service.OutboxRepository,
	publisher service.MessagePublisher,
	metrics service.OutboxMetrics,
	logger *slog.Logger,
) *service.OutboxRelay {
	return service.NewOutboxRelay(repository, publisher, outboxRelayConfig(serviceConfig), metrics, logger)
}

func outboxRelayConfig(serviceConfig config.Config) service.OutboxRelayConfig {
	return service.OutboxRelayConfig{
		PollInterval:    serviceConfig.OutboxInterval,
		MaxAttempts:     serviceConfig.OutboxAttempts,
		CleanupInterval: serviceConfig.OutboxCleanupInterval,
		Retention:       serviceConfig.OutboxRetention,
	}
}
