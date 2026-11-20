package bootstrap

import (
	"context"
	"log/slog"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/config"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/tracing"
)

func startTracing(serviceConfig config.Config, logger *slog.Logger) func() {
	stop := tracing.Start(
		context.Background(),
		tracing.Config{ServiceName: config.ServiceName, Endpoint: serviceConfig.OTLPEndpoint},
		logger,
	)

	return func() {
		stopContext, cancel := context.WithTimeout(context.Background(), serviceConfig.ShutdownTimeout)
		defer cancel()

		stop(stopContext)
	}
}
