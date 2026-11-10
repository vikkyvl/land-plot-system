package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/logging"
)

const (
	consumerTagPrefix = "report-service-"
	reconnectDelay    = 2 * time.Second
	redeliveryDelay   = time.Second
)

type EventHandler interface {
	Handle(processingContext context.Context, event domain.PlotEvent) error
}

type ConsumerConfig struct {
	URL               string
	ConnectionTimeout time.Duration
	ProcessingTimeout time.Duration
	Prefetch          int
	InstanceID        string
}

type Consumer struct {
	consumerConfig ConsumerConfig
	logger         *slog.Logger
}

func NewConsumer(consumerConfig ConsumerConfig, logger *slog.Logger) *Consumer {
	return &Consumer{consumerConfig: consumerConfig, logger: logger}
}

func (consumer *Consumer) Run(runContext context.Context, handler EventHandler) {
	for runContext.Err() == nil {
		if err := consumer.consume(runContext, handler); err != nil {
			consumer.logger.Warn(logPlotEventConsumerDisconnected, logging.KeyError, err)
		}

		select {
		case <-runContext.Done():
		case <-time.After(reconnectDelay):
		}
	}
}

func (consumer *Consumer) consume(runContext context.Context, handler EventHandler) error {
	connection, channel, err := openChannel(consumer.consumerConfig.URL, consumer.consumerConfig.ConnectionTimeout)
	if err != nil {
		return err
	}
	defer func() { _ = closeChannel(connection, channel) }()

	if err := declareTopology(channel); err != nil {
		return err
	}

	if err := channel.Qos(consumer.consumerConfig.Prefetch, 0, false); err != nil {
		return fmt.Errorf(wrapConfigureConsumerPrefetch, err)
	}

	deliveries, err := channel.ConsumeWithContext(
		runContext,
		queueName,
		consumerTagPrefix+consumer.consumerConfig.InstanceID,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf(wrapStartPlotEventConsumer, err)
	}

	consumer.logger.Info(logPlotEventConsumerStarted, logging.KeyQueue, queueName)

	for {
		select {
		case <-runContext.Done():
			return nil
		case delivery, ok := <-deliveries:
			if !ok {
				return errDeliveriesClosed
			}

			if err := consumer.handleDelivery(runContext, delivery, handler); err != nil {
				return err
			}
		}
	}
}

func (consumer *Consumer) handleDelivery(
	runContext context.Context,
	delivery amqp.Delivery,
	handler EventHandler,
) error {
	event, err := decodePlotMessage(delivery.Body, delivery.RoutingKey)
	if err != nil {
		consumer.logger.Warn(
			logRejectInvalidPlotEvent,
			logging.KeyRoutingKey, delivery.RoutingKey,
			logging.KeyError, err,
		)

		return consumer.reject(delivery)
	}

	processingContext, cancel := context.WithTimeout(runContext, consumer.consumerConfig.ProcessingTimeout)
	defer cancel()

	if err := handler.Handle(processingContext, event); err != nil {
		if errors.Is(err, domain.ErrRejectedEvent) {
			consumer.logger.Error(
				logProcessPlotEvent,
				logging.KeyEventID, event.ID,
				logging.KeyEventType, event.Type,
				logging.KeyPlotID, event.PlotID,
				logging.KeyError, err,
			)

			return consumer.reject(delivery)
		}

		return consumer.requeueLater(runContext, delivery, event, err)
	}

	if err := delivery.Ack(false); err != nil {
		return fmt.Errorf(wrapAcknowledgePlotEvent, err)
	}

	consumer.logger.Info(
		logPlotEventProcessed,
		logging.KeyEventID, event.ID,
		logging.KeyEventType, event.Type,
		logging.KeyPlotID, event.PlotID,
		logging.KeyVersion, event.Version,
	)

	return nil
}

func (consumer *Consumer) requeueLater(
	runContext context.Context,
	delivery amqp.Delivery,
	event domain.PlotEvent,
	cause error,
) error {
	consumer.logger.Warn(
		logRequeuePlotEvent,
		logging.KeyEventID, event.ID,
		logging.KeyEventType, event.Type,
		logging.KeyPlotID, event.PlotID,
		logging.KeyError, cause,
	)

	select {
	case <-runContext.Done():
	case <-time.After(redeliveryDelay):
	}

	if err := delivery.Nack(false, true); err != nil {
		return fmt.Errorf(wrapRequeuePlotEvent, err)
	}

	return nil
}

func (consumer *Consumer) reject(delivery amqp.Delivery) error {
	if err := delivery.Reject(false); err != nil {
		return fmt.Errorf(wrapRejectPlotEvent, err)
	}

	return nil
}
