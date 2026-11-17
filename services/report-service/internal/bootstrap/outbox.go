package bootstrap

import (
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/config"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/service"
)

func outboxRelayConfig(serviceConfig config.Config) service.OutboxRelayConfig {
	return service.OutboxRelayConfig{
		PollInterval:    serviceConfig.OutboxInterval,
		MaxAttempts:     serviceConfig.OutboxAttempts,
		CleanupInterval: serviceConfig.OutboxCleanupInterval,
		Retention:       serviceConfig.OutboxRetention,
	}
}
