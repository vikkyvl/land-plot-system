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

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/config"
	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/handler"
	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/logging"
	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/metrics"
	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/repository/postgres"
	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/service"
)

func Run() error {
	serviceConfig, err := config.Load()
	if err != nil {
		return fmt.Errorf(wrapLoadConfig, err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With(logging.KeyService, config.ServiceName)

	pool, err := openDatabase(serviceConfig.DatabaseURL, serviceConfig.DatabaseTimeout)
	if err != nil {
		return err
	}
	defer pool.Close()

	userRepository := postgres.NewUserRepository(pool)
	if err := migrate(userRepository, serviceConfig.DatabaseTimeout); err != nil {
		return err
	}

	serviceMetrics := metrics.New(config.ServiceName)
	identity := service.NewIdentityService(userRepository, serviceMetrics)

	verifier, stopVerifier, err := newTokenVerifier(serviceConfig, logger)
	if err != nil {
		return err
	}
	defer stopVerifier()

	router := handler.NewRouter(logger, config.ServiceName, identity, verifier)
	server := &http.Server{Handler: serviceMetrics.ObserveRequests(router), ReadHeaderTimeout: readHeaderTimeout}

	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", serviceConfig.HTTPAddress)
	if err != nil {
		return fmt.Errorf(wrapListenHTTP, err)
	}

	metricsEndpoint, err := startMetricsServer(serviceConfig.MetricsAddress, serviceMetrics.Handler(), logger)
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
