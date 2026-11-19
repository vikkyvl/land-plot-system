package bootstrap

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/vleshchenko/land-plot-system/ai/internal/config"
	"github.com/vleshchenko/land-plot-system/ai/internal/logging"
	"github.com/vleshchenko/land-plot-system/ai/internal/tracing"
)

func startTracing(serviceName string, logger *slog.Logger) func() {
	stop := tracing.Start(
		context.Background(),
		tracing.Config{ServiceName: serviceName, Endpoint: config.LoadOTLPEndpoint()},
		logger,
	)

	return func() {
		stopContext, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()

		stop(stopContext)
	}
}

func tracedHandler(logger *slog.Logger, handler http.Handler) http.Handler {
	named := tracing.NameByRoute(logging.PatternRoute, handler)

	return tracing.Server(logging.AccessLog(logger, logging.PatternRoute, named))
}
