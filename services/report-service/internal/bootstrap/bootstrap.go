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
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/repository/postgres"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/service"
)

func Run() error {
	serviceConfig, err := config.Load()
	if err != nil {
		return fmt.Errorf(wrapLoadConfig, err)
	}

	logger := logging.NewLogger(os.Stdout, config.ServiceName, slog.LevelInfo)
	defer startTracing(serviceConfig, logger)()

	pool, err := openDatabase(serviceConfig.DatabaseURL, serviceConfig.DatabaseTimeout)
	if err != nil {
		return err
	}
	defer pool.Close()

	outboxRepository, serviceMetrics, faultSwitch := newMetrics(serviceConfig, pool, logger)

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
	server := newHTTPServer(
		faultSwitch.GuardHealth(handler.NewRouter(logger, config.ServiceName, reports, exports, verifier)),
		serviceMetrics,
		logger,
	)

	listener, metricsEndpoint, err := listen(serviceConfig, serviceMetrics.Handler(), faultSwitch, logger)
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
