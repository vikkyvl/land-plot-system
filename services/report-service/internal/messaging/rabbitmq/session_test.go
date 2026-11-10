package rabbitmq

import (
	"errors"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

func TestSessionReportsMessageReturnedAsUnroutable(t *testing.T) {
	returns := make(chan amqp.Return, 1)
	returns <- amqp.Return{MessageId: "event-1", RoutingKey: "plot.created"}

	err := (&amqpSession{returns: returns}).checkRouted("event-1")
	if !errors.Is(err, errMessageUnroutable) || !errors.Is(err, domain.ErrMessageRejected) {
		t.Fatalf("checkRouted() error = %v, want rejected unroutable message", err)
	}
}

func TestSessionStopsReadingReturnsAfterConnectionLoss(t *testing.T) {
	returns := make(chan amqp.Return)
	close(returns)

	session := &amqpSession{returns: returns}
	session.discardStaleReturns()

	if err := session.checkRouted("event-1"); err != nil {
		t.Fatalf("checkRouted() error = %v, want nil for closed return channel", err)
	}
}
