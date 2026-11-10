package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const returnedMessagesBuffer = 16

type brokerSession interface {
	publish(publishContext context.Context, routingKey string, message amqp.Publishing) error
	isClosed() bool
	close() error
}

type sessionDialer func() (brokerSession, error)

type amqpSession struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	returns    chan amqp.Return
}

func amqpDialer(brokerURL string, timeout time.Duration) sessionDialer {
	return func() (brokerSession, error) {
		return openSession(brokerURL, timeout)
	}
}

func openSession(brokerURL string, timeout time.Duration) (*amqpSession, error) {
	connection, channel, err := openChannel(brokerURL, timeout)
	if err != nil {
		return nil, err
	}

	session := &amqpSession{
		connection: connection,
		channel:    channel,
		returns:    channel.NotifyReturn(make(chan amqp.Return, returnedMessagesBuffer)),
	}
	if err := session.setup(); err != nil {
		_ = session.close()
		return nil, err
	}

	return session, nil
}

func openChannel(brokerURL string, timeout time.Duration) (*amqp.Connection, *amqp.Channel, error) {
	connection, err := amqp.DialConfig(brokerURL, amqp.Config{Dial: amqp.DefaultDial(timeout)})
	if err != nil {
		return nil, nil, fmt.Errorf(wrapConnectToRabbitMQ, err)
	}

	channel, err := connection.Channel()
	if err != nil {
		_ = connection.Close()
		return nil, nil, fmt.Errorf(wrapOpenRabbitMQChannel, err)
	}

	return connection, channel, nil
}

func (session *amqpSession) setup() error {
	if err := declareExchange(session.channel, exchangeName); err != nil {
		return err
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
	session.discardStaleReturns()

	confirmation, err := session.channel.PublishWithDeferredConfirmWithContext(
		publishContext,
		exchangeName,
		routingKey,
		true,
		false,
		message,
	)
	if err != nil {
		return fmt.Errorf(wrapPublishMessage, err)
	}

	acknowledged, err := confirmation.WaitContext(publishContext)
	if err != nil {
		return fmt.Errorf(wrapWaitForPublisherConfirmation, err)
	}

	if !acknowledged {
		return errBrokerRejectedMessage
	}

	return session.checkRouted(message.MessageId)
}

func (session *amqpSession) checkRouted(messageID string) error {
	for {
		select {
		case returned, open := <-session.returns:
			if !open {
				return nil
			}

			if returned.MessageId == messageID {
				return fmt.Errorf(wrapMessageNotRouted, errMessageUnroutable, returned.RoutingKey)
			}
		default:
			return nil
		}
	}
}

func (session *amqpSession) discardStaleReturns() {
	for {
		select {
		case _, open := <-session.returns:
			if !open {
				return
			}
		default:
			return
		}
	}
}

func (session *amqpSession) isClosed() bool {
	return session.connection.IsClosed() || session.channel.IsClosed()
}

func (session *amqpSession) close() error {
	return closeChannel(session.connection, session.channel)
}

func closeChannel(connection *amqp.Connection, channel *amqp.Channel) error {
	if err := channel.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) {
		_ = connection.Close()
		return fmt.Errorf(wrapCloseRabbitMQChannel, err)
	}

	if err := connection.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) {
		return fmt.Errorf(wrapCloseRabbitMQConnection, err)
	}

	return nil
}

func declareExchange(channel *amqp.Channel, name string) error {
	if err := channel.ExchangeDeclare(name, amqp.ExchangeTopic, true, false, false, false, nil); err != nil {
		return fmt.Errorf(wrapDeclareExchange, name, err)
	}

	return nil
}
