package bootstrap

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/config"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/handler"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/logging"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/messaging/rabbitmq"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/repository/postgres"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/service"
)

func Run() error {
	serviceConfig, accuracy, err := loadConfig()
	if err != nil {
		return err
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

	outboxRepository, serviceMetrics := newMetrics(serviceConfig, pool, plotRepository)

	publisher, closePublisher := openPublisher(serviceConfig, logger)
	defer closePublisher()

	plots := newPlotService(plotRepository, accuracy, serviceConfig.SagaTimeout, serviceMetrics)
	sagas := service.NewSagaService(plotRepository, serviceMetrics, logger)
	relay := service.NewOutboxRelay(
		outboxRepository,
		publisher,
		outboxRelayConfig(serviceConfig),
		serviceMetrics,
		logger,
	)
	watcher := service.NewSagaTimeoutWatcher(sagas, logger)
	consumer := rabbitmq.NewConsumer(rabbitmq.ConsumerConfig{
		URL:               serviceConfig.RabbitMQURL,
		ConnectionTimeout: serviceConfig.RabbitMQTimeout,
		ProcessingTimeout: serviceConfig.ProcessingTimeout,
		InstanceID:        serviceConfig.InstanceID,
	}, serviceMetrics, logger)

	verifier, stopVerifier, err := newTokenVerifier(serviceConfig, logger)
	if err != nil {
		return err
	}
	defer stopVerifier()

	router := handler.NewRouter(logger, config.ServiceName, serviceConfig.InstanceID, plots, verifier)
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
		watcher.Run,
		func(workerContext context.Context) { consumer.Run(workerContext, sagas) },
	)
	serveErr := serve(runContext, server, listener, serviceConfig.ShutdownTimeout, instance.deregister, logger)

	stop()
	waitWorkers()

	return serveErr
}
