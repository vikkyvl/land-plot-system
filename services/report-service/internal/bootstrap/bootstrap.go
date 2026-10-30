package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/auth"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/client/httpclient"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/config"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/discovery/consul"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/handler"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/logging"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/messaging/rabbitmq"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/repository/postgres"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/service"
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

	consumer, err := rabbitmq.NewConsumer(rabbitmq.ConsumerConfig{
		URL:               serviceConfig.RabbitMQURL,
		ConnectionTimeout: serviceConfig.RabbitMQTimeout,
		ProcessingTimeout: serviceConfig.ProcessingTimeout,
		Prefetch:          serviceConfig.EventPrefetch,
		InstanceID:        serviceConfig.InstanceID,
	}, logger)
	if err != nil {
		return fmt.Errorf(wrapCreatePlotEventConsumer, err)
	}
	defer func() {
		if err := consumer.Close(); err != nil {
			logger.Error(logClosePlotEventConsumer, logging.KeyError, err)
		}
	}()

	resolver := consul.NewResolver(serviceConfig.ConsulAddress, &http.Client{Timeout: serviceConfig.ConsulTimeout})
	plotClient := httpclient.NewPlotClient(
		resolver,
		&http.Client{Timeout: serviceConfig.UpstreamTimeout},
		serviceConfig.PlotServiceName,
	)
	reports := service.NewReportService(postgres.NewReportRepository(pool), plotClient, serviceConfig.InstanceID)
	projection := service.NewPlotProjection(postgres.NewPlotSummaryRepository(pool))
	router := handler.NewRouter(logger, config.ServiceName, reports, auth.NewJWTVerifier(serviceConfig.JWTSecret))
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

	waitConsumer := startConsumer(runContext, consumer, projection, stop)
	serveErr := serve(runContext, server, listener, serviceConfig.ShutdownTimeout, instance.deregister, logger)

	stop()

	return errors.Join(serveErr, waitConsumer())
}
