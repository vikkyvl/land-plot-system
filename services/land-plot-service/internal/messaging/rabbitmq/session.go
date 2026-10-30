package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type brokerSession interface {
	publish(publishContext context.Context, routingKey string, message amqp.Publishing) error
	isClosed() bool
	close() error
}

type sessionDialer func() (brokerSession, error)

type amqpSession struct {
	connection *amqp.Connection
	channel    *amqp.Channel
}

func amqpDialer(brokerURL string, timeout time.Duration) sessionDialer {
	return func() (brokerSession, error) {
		return openSession(brokerURL, timeout)
	}
}

func openSession(brokerURL string, timeout time.Duration) (*amqpSession, error) {
	connection, err := amqp.DialConfig(brokerURL, amqp.Config{Dial: amqp.DefaultDial(timeout)})
	if err != nil {
		return nil, fmt.Errorf(wrapConnectToRabbitMQ, err)
	}

	channel, err := connection.Channel()
	if err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf(wrapOpenRabbitMQChannel, err)
	}

	session := &amqpSession{connection: connection, channel: channel}
	if err := session.setup(); err != nil {
		_ = session.close()
		return nil, err
	}

	return session, nil
}

func (session *amqpSession) setup() error {
	if err := session.channel.ExchangeDeclare(
		exchangeName,
		amqp.ExchangeTopic,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf(wrapDeclareEventExchange, err)
	}

	if err := session.channel.Confirm(false); err != nil {
		return fmt.Errorf(wrapEnablePublisherConfirms, err)
	}

	return nil
}

func (session *amqpSession) publish(
	publishContext context.Context,
	routingKey string,
	message amqp.Publishing,
) error {
	confirmation, err := session.channel.PublishWithDeferredConfirmWithContext(
		publishContext,
		exchangeName,
		routingKey,
		false,
		false,
		message,
	)
	if err != nil {
		return fmt.Errorf(wrapPublishPlotEvent, err)
	}

	acknowledged, err := confirmation.WaitContext(publishContext)
	if err != nil {
		return fmt.Errorf(wrapWaitForPublisherConfirmation, err)
	}

	if !acknowledged {
		return errBrokerRejectedPlotEvent
	}

	return nil
}

func (session *amqpSession) isClosed() bool {
	return session.connection.IsClosed() || session.channel.IsClosed()
}

func (session *amqpSession) close() error {
	if err := session.channel.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) {
		_ = session.connection.Close()
		return fmt.Errorf(wrapCloseRabbitMQChannel, err)
	}

	if err := session.connection.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) {
		return fmt.Errorf(wrapCloseRabbitMQConnection, err)
	}

	return nil
}
