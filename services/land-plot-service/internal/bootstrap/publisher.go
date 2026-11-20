package bootstrap

import (
	"log/slog"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/config"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/logging"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/messaging/rabbitmq"
)

func openPublisher(serviceConfig config.Config, logger *slog.Logger) (*rabbitmq.Publisher, func()) {
	publisher := rabbitmq.NewPublisher(serviceConfig.RabbitMQURL, serviceConfig.RabbitMQTimeout)

	return publisher, func() {
		if err := publisher.Close(); err != nil {
			logger.Error(logCloseEventPublisher, logging.KeyError, err)
		}
	}
}
