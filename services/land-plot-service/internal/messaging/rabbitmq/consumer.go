package rabbitmq

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/logging"
)

const (
	consumerTagPrefix = "land-plot-service-"
	sagaPrefetch      = 10
	reconnectDelay    = 2 * time.Second
	redeliveryDelay   = time.Second
)

type SagaHandler interface {
	Confirm(processingContext context.Context, sagaID string) error
	Compensate(processingContext context.Context, sagaID string, reason domain.CompensationReason) error
}

type ConsumerConfig struct {
	URL               string
	ConnectionTimeout time.Duration
	ProcessingTimeout time.Duration
	InstanceID        string
}

type Consumer struct {
	consumerConfig ConsumerConfig
	observer       DeliveryObserver
	logger         *slog.Logger
}

func NewConsumer(consumerConfig ConsumerConfig, observer DeliveryObserver, logger *slog.Logger) *Consumer {
	return &Consumer{consumerConfig: consumerConfig, observer: observer, logger: logger}
}

func (consumer *Consumer) Run(runContext context.Context, handler SagaHandler) {
	for runContext.Err() == nil {
		if err := consumer.consume(runContext, handler); err != nil {
			consumer.logger.Warn(logSagaConsumerDisconnected, logging.KeyError, err)
		}

		select {
		case <-runContext.Done():
		case <-time.After(reconnectDelay):
		}
	}
}

func (consumer *Consumer) consume(runContext context.Context, handler SagaHandler) error {
	connection, channel, err := openChannel(consumer.consumerConfig.URL, consumer.consumerConfig.ConnectionTimeout)
	if err != nil {
		return err
	}
	defer func() { _ = closeChannel(connection, channel) }()

	if err := declareSagaTopology(channel); err != nil {
		return err
	}

	if err := channel.Qos(sagaPrefetch, 0, false); err != nil {
		return fmt.Errorf(wrapConfigureConsumerPrefetch, err)
	}

	deliveries, err := channel.ConsumeWithContext(
		runContext,
		sagaQueueName,
		consumerTagPrefix+consumer.consumerConfig.InstanceID,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf(wrapStartSagaConsumer, err)
	}

	consumer.logger.Info(logSagaConsumerStarted, logging.KeyQueue, sagaQueueName)

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
	handler SagaHandler,
) error {
	deliveryContext := logging.WithRequestID(runContext, requestID(delivery))

	processingContext, cancel := context.WithTimeout(deliveryContext, consumer.consumerConfig.ProcessingTimeout)
	defer cancel()

	valid, err := consumer.dispatch(processingContext, delivery, handler)
	if err != nil {
		consumer.observer.DeliveryProcessed(sagaQueueName, OutcomeRequeued, createdAt(delivery))
		return consumer.retryLater(deliveryContext, delivery, err)
	}

	if !valid {
		consumer.observer.DeliveryProcessed(sagaQueueName, OutcomeRejected, createdAt(delivery))
		return reject(delivery)
	}

	consumer.observer.DeliveryProcessed(sagaQueueName, OutcomeAcknowledged, createdAt(delivery))

	if err := delivery.Ack(false); err != nil {
		return fmt.Errorf(wrapAcknowledgeSagaMessage, err)
	}

	return nil
}

func (consumer *Consumer) dispatch(
	processingContext context.Context,
	delivery amqp.Delivery,
	handler SagaHandler,
) (bool, error) {
	if delivery.Exchange == deadExchangeName {
		if !isUUID(delivery.MessageId) {
			consumer.logInvalid(processingContext, delivery, errInvalidEventID)
			return false, nil
		}

		return true, handler.Compensate(processingContext, delivery.MessageId, domain.ReasonDeadLetter)
	}

	message, err := decodeSummaryApplied(delivery.Body)
	if err != nil {
		consumer.logInvalid(processingContext, delivery, err)
		return false, nil
	}

	return true, handler.Confirm(processingContext, message.AppliedEventID)
}

func (consumer *Consumer) retryLater(runContext context.Context, delivery amqp.Delivery, cause error) error {
	consumer.logger.ErrorContext(
		runContext,
		logProcessSagaMessage,
		logging.KeyMessageID, delivery.MessageId,
		logging.KeyRoutingKey, delivery.RoutingKey,
		logging.KeyError, cause,
	)

	select {
	case <-runContext.Done():
	case <-time.After(redeliveryDelay):
	}

	if err := delivery.Nack(false, true); err != nil {
		return fmt.Errorf(wrapRequeueSagaMessage, err)
	}

	return nil
}

func (consumer *Consumer) logInvalid(logContext context.Context, delivery amqp.Delivery, cause error) {
	consumer.logger.WarnContext(
		logContext,
		logRejectInvalidSagaMessage,
		logging.KeyMessageID, delivery.MessageId,
		logging.KeyRoutingKey, delivery.RoutingKey,
		logging.KeyError, cause,
	)
}

func reject(delivery amqp.Delivery) error {
	if err := delivery.Reject(false); err != nil {
		return fmt.Errorf(wrapRejectSagaMessage, err)
	}

	return nil
}
