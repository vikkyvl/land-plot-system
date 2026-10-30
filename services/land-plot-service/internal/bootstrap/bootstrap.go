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

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/auth"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/config"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/handler"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/logging"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/messaging/rabbitmq"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/repository/postgres"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/service"
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

	plotRepository := postgres.NewPlotRepository(pool)
	if err := migrate(plotRepository, serviceConfig.DatabaseTimeout); err != nil {
		return err
	}

	publisher, err := rabbitmq.NewPublisher(serviceConfig.RabbitMQURL, serviceConfig.RabbitMQTimeout)
	if err != nil {
		return fmt.Errorf(wrapCreateEventPublisher, err)
	}
	defer func() {
		if err := publisher.Close(); err != nil {
			logger.Error(logCloseEventPublisher, logging.KeyError, err)
		}
	}()

	plots := service.NewPlotService(plotRepository, publisher, domain.LocalEllipsoidCalculator{}, logger)
	router := handler.NewRouter(
		logger,
		config.ServiceName,
		serviceConfig.InstanceID,
		plots,
		auth.NewJWTVerifier(serviceConfig.JWTSecret),
	)
	server := &http.Server{Handler: router, ReadHeaderTimeout: readHeaderTimeout}

	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", serviceConfig.HTTPAddress)
	if err != nil {
		return fmt.Errorf(wrapListenHTTP, err)
	}

	instance, err := register(serviceConfig, logger)
	if err != nil {
		_ = listener.Close()
		return err
	}

	runContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return serve(runContext, server, listener, serviceConfig.ShutdownTimeout, instance.deregister, logger)
}
