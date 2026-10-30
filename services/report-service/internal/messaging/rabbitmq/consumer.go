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

const consumerTagPrefix = "report-service-"

var errDeliveriesClosed = errors.New("plot event delivery channel closed")

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
	connection        *amqp.Connection
	channel           *amqp.Channel
	consumerTag       string
	processingTimeout time.Duration
	logger            *slog.Logger
}

func NewConsumer(serviceConfig ConsumerConfig, logger *slog.Logger) (*Consumer, error) {
	connection, err := amqp.DialConfig(
		serviceConfig.URL,
		amqp.Config{Dial: amqp.DefaultDial(serviceConfig.ConnectionTimeout)},
	)
	if err != nil {
		return nil, fmt.Errorf(wrapConnectToRabbitMQ, err)
	}

	channel, err := connection.Channel()
	if err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf(wrapOpenRabbitMQChannel, err)
	}

	consumer := &Consumer{
		connection:        connection,
		channel:           channel,
		consumerTag:       consumerTagPrefix + serviceConfig.InstanceID,
		processingTimeout: serviceConfig.ProcessingTimeout,
		logger:            logger,
	}
	if err := declareTopology(channel); err != nil {
		_ = consumer.Close()
		return nil, err
	}

	if err := channel.Qos(serviceConfig.Prefetch, 0, false); err != nil {
		_ = consumer.Close()
		return nil, fmt.Errorf(wrapConfigureConsumerPrefetch, err)
	}

	return consumer, nil
}

func (consumer *Consumer) Run(runContext context.Context, handler EventHandler) error {
	deliveries, err := consumer.channel.ConsumeWithContext(
		runContext,
		queueName,
		consumer.consumerTag,
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
				return deliveriesClosed(runContext)
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

	processingContext, cancel := context.WithTimeout(runContext, consumer.processingTimeout)
	defer cancel()

	if err := handler.Handle(processingContext, event); err != nil {
		consumer.logger.Error(
			logProcessPlotEvent,
			logging.KeyEventID, event.ID,
			logging.KeyEventType, event.Type,
			logging.KeyError, err,
		)

		return consumer.reject(delivery)
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

func (consumer *Consumer) reject(delivery amqp.Delivery) error {
	if err := delivery.Reject(false); err != nil {
		return fmt.Errorf(wrapRejectPlotEvent, err)
	}

	return nil
}

func (consumer *Consumer) Close() error {
	if err := consumer.channel.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) {
		_ = consumer.connection.Close()
		return fmt.Errorf(wrapCloseRabbitMQChannel, err)
	}

	if err := consumer.connection.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) {
		return fmt.Errorf(wrapCloseRabbitMQConnection, err)
	}

	return nil
}

func deliveriesClosed(runContext context.Context) error {
	select {
	case <-runContext.Done():
		return nil
	default:
		return errDeliveriesClosed
	}
}
