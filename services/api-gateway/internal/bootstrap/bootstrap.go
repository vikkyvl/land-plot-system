package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/balancer"
	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/config"
	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/discovery/consul"
	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/handler"
	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/logging"
	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/metrics"
	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/proxy"
	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/ratelimit"
)

func Run() error {
	serviceConfig, err := config.Load()
	if err != nil {
		return fmt.Errorf(wrapLoadConfig, err)
	}

	logger := logging.NewLogger(os.Stdout, config.ServiceName, slog.LevelInfo)
	gatewayMetrics := metrics.NewGateway(config.ServiceName)

	resolver := consul.NewResolver(serviceConfig.ConsulAddress, &http.Client{Timeout: serviceConfig.ConsulTimeout})
	gateway := proxy.NewGateway(
		resolver,
		balancer.NewRoundRobin(),
		logger,
		serviceConfig.UpstreamTimeout,
		serviceConfig.InstanceID,
		gatewayMetrics,
	)

	verifier, stopVerifier, err := newTokenVerifier(serviceConfig, logger)
	if err != nil {
		return err
	}
	defer stopVerifier()

	router := handler.NewRouter(
		logger,
		config.ServiceName,
		gateway,
		verifier,
		ratelimit.NewIPLimiter(serviceConfig.PreviewRateLimitPerMinute),
		serviceConfig.AllowedOrigins,
		gatewayMetrics,
	)
	server := &http.Server{Handler: router, ReadHeaderTimeout: readHeaderTimeout}

	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", serviceConfig.HTTPAddress)
	if err != nil {
		return fmt.Errorf(wrapListenHTTP, err)
	}

	metricsEndpoint, err := startMetricsServer(serviceConfig.MetricsAddress, gatewayMetrics.Handler(), logger)
	if err != nil {
		_ = listener.Close()
		return err
	}
	defer metricsEndpoint.stop(serviceConfig.ShutdownTimeout)

	instance, err := register(serviceConfig, logger)
	if err != nil {
		_ = listener.Close()
		return err
	}

	runContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return serve(runContext, server, listener, serviceConfig.ShutdownTimeout, instance.deregister, logger)
}
