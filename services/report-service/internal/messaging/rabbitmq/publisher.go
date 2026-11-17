package rabbitmq

import (
	"context"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

type Publisher struct {
	session brokerSession
	dial    sessionDialer
	timeout time.Duration
	mutex   sync.Mutex
}

func NewPublisher(brokerURL string, timeout time.Duration) *Publisher {
	return &Publisher{dial: amqpDialer(brokerURL, timeout), timeout: timeout}
}

func (publisher *Publisher) Publish(publishContext context.Context, outboxMessage domain.OutboxMessage) error {
	message := amqp.Publishing{
		ContentType:  contentTypeJSON,
		DeliveryMode: amqp.Persistent,
		MessageId:    outboxMessage.ID,
		Type:         outboxMessage.RoutingKey,
		Body:         outboxMessage.Payload,
	}

	publisher.mutex.Lock()
	defer publisher.mutex.Unlock()

	if publisher.session == nil {
		if err := publisher.reconnect(); err != nil {
			return err
		}
	}

	err := publisher.publishOnce(publishContext, outboxMessage.RoutingKey, message)
	if err == nil || !publisher.session.isClosed() {
		return err
	}

	if err := publisher.reconnect(); err != nil {
		return err
	}

	return publisher.publishOnce(publishContext, outboxMessage.RoutingKey, message)
}

func (publisher *Publisher) publishOnce(
	requestContext context.Context,
	routingKey string,
	message amqp.Publishing,
) error {
	publishContext, cancel := context.WithTimeout(requestContext, publisher.timeout)
	defer cancel()

	return publisher.session.publish(publishContext, routingKey, message)
}

func (publisher *Publisher) reconnect() error {
	session, err := publisher.dial()
	if err != nil {
		return fmt.Errorf(wrapReconnectToRabbitMQ, err)
	}

	if publisher.session != nil {
		_ = publisher.session.close()
	}

	publisher.session = session

	return nil
}

func (publisher *Publisher) Close() error {
	publisher.mutex.Lock()
	defer publisher.mutex.Unlock()

	if publisher.session == nil {
		return nil
	}

	return publisher.session.close()
}
