package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/config"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/document"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/handler"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/logging"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/messaging/rabbitmq"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/metrics"
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

	reportRepository := postgres.NewReportRepository(pool)
	outboxRepository := postgres.NewOutboxRepository(pool)
	serviceMetrics := metrics.New(config.ServiceName, metrics.Stats{
		Backlog: outboxRepository,
		Reports: reportRepository,
		Timeout: serviceConfig.DatabaseTimeout,
	})

	consumer := rabbitmq.NewConsumer(rabbitmq.ConsumerConfig{
		URL:               serviceConfig.RabbitMQURL,
		ConnectionTimeout: serviceConfig.RabbitMQTimeout,
		ProcessingTimeout: serviceConfig.ProcessingTimeout,
		Prefetch:          serviceConfig.EventPrefetch,
		InstanceID:        serviceConfig.InstanceID,
	}, serviceMetrics, logger)

	publisher, closePublisher := openPublisher(serviceConfig, logger)
	defer closePublisher()

	reports := newReportService(serviceConfig, pool, serviceMetrics, logger)
	projection := service.NewPlotProjection(postgres.NewPlotSummaryRepository(pool, serviceConfig.FailNamePrefix))
	relay := service.NewOutboxRelay(
		outboxRepository,
		publisher,
		outboxRelayConfig(serviceConfig),
		serviceMetrics,
		logger,
	)

	verifier, stopVerifier, err := newTokenVerifier(serviceConfig, logger)
	if err != nil {
		return err
	}
	defer stopVerifier()

	exports := service.NewExportService(reports, document.Renderer{}, serviceMetrics)
	router := handler.NewRouter(logger, config.ServiceName, reports, exports, verifier)
	server := newHTTPServer(serviceMetrics.ObserveRequests(router))

	listener, metricsEndpoint, err := listen(serviceConfig, serviceMetrics.Handler(), logger)
	if err != nil {
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

	waitWorkers := startWorkers(
		runContext,
		relay.Run,
		func(workerContext context.Context) { consumer.Run(workerContext, projection) },
	)
	serveErr := serve(runContext, server, listener, serviceConfig.ShutdownTimeout, instance.deregister, logger)

	stop()
	waitWorkers()

	return serveErr
}
