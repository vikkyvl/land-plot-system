package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/config"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/faults"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/logging"
)

const metricsPath = "/metrics"

type metricsServer struct {
	server *http.Server
	logger *slog.Logger
}

func startMetricsServer(
	address string,
	metrics http.Handler,
	faultSwitch *faults.Switch,
	logger *slog.Logger,
) (*metricsServer, error) {
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", address)
	if err != nil {
		return nil, fmt.Errorf(wrapListenMetrics, err)
	}

	router := http.NewServeMux()
	router.Handle(http.MethodGet+" "+metricsPath, metrics)
	faultSwitch.Register(router)

	server := &http.Server{Handler: router, ReadHeaderTimeout: readHeaderTimeout}

	go func() {
		logger.Info(logMetricsServerStarted, logging.KeyAddress, listener.Addr().String())

		if err := server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
			logger.Error(logMetricsServerFailed, logging.KeyError, err)
		}
	}()

	return &metricsServer{server: server, logger: logger}, nil
}

func listen(
	serviceConfig config.Config,
	metrics http.Handler,
	faultSwitch *faults.Switch,
	logger *slog.Logger,
) (net.Listener, *metricsServer, error) {
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", serviceConfig.HTTPAddress)
	if err != nil {
		return nil, nil, fmt.Errorf(wrapListenHTTP, err)
	}

	metricsEndpoint, err := startMetricsServer(serviceConfig.MetricsAddress, metrics, faultSwitch, logger)
	if err != nil {
		_ = listener.Close()
		return nil, nil, err
	}

	return listener, metricsEndpoint, nil
}

func (metrics *metricsServer) stop(timeout time.Duration) {
	shutdownContext, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := metrics.server.Shutdown(shutdownContext); err != nil {
		metrics.logger.Error(logMetricsServerFailed, logging.KeyError, err)
	}
}
