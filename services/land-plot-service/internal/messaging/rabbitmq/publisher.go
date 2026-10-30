package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

type Publisher struct {
	session brokerSession
	dial    sessionDialer
	timeout time.Duration
	mutex   sync.Mutex
}

func NewPublisher(brokerURL string, timeout time.Duration) (*Publisher, error) {
	dial := amqpDialer(brokerURL, timeout)

	session, err := dial()
	if err != nil {
		return nil, err
	}

	return &Publisher{session: session, dial: dial, timeout: timeout}, nil
}

func (publisher *Publisher) Publish(requestContext context.Context, event domain.PlotEvent) error {
	body, err := json.Marshal(newPlotMessage(event))
	if err != nil {
		return fmt.Errorf(wrapEncodePlotEvent, err)
	}

	message := amqp.Publishing{
		ContentType:  contentTypeJSON,
		DeliveryMode: amqp.Persistent,
		MessageId:    event.ID,
		Type:         string(event.Type),
		Timestamp:    event.OccurredAt,
		Body:         body,
	}

	publisher.mutex.Lock()
	defer publisher.mutex.Unlock()

	err = publisher.publishOnce(requestContext, string(event.Type), message)
	if err == nil || !publisher.session.isClosed() {
		return err
	}

	if err := publisher.reconnect(); err != nil {
		return err
	}

	return publisher.publishOnce(requestContext, string(event.Type), message)
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

	_ = publisher.session.close()
	publisher.session = session

	return nil
}

func (publisher *Publisher) Close() error {
	publisher.mutex.Lock()
	defer publisher.mutex.Unlock()

	return publisher.session.close()
}
