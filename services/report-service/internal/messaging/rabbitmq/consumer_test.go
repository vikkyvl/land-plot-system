package rabbitmq

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

type acknowledgerStub struct {
	acked    bool
	rejected bool
	requeue  bool
}

func (stub *acknowledgerStub) Ack(uint64, bool) error {
	stub.acked = true
	return nil
}

func (stub *acknowledgerStub) Nack(uint64, bool, bool) error { return nil }

func (stub *acknowledgerStub) Reject(_ uint64, requeue bool) error {
	stub.rejected = true
	stub.requeue = requeue

	return nil
}

func (stub *acknowledgerStub) deadLettered() bool {
	return stub.rejected && !stub.requeue && !stub.acked
}

type eventHandlerStub struct {
	event domain.PlotEvent
	err   error
}

func (stub *eventHandlerStub) Handle(_ context.Context, event domain.PlotEvent) error {
	stub.event = event
	return stub.err
}

func testDelivery(body []byte, routingKey string) (amqp.Delivery, *acknowledgerStub) {
	acknowledger := &acknowledgerStub{}
	return amqp.Delivery{Acknowledger: acknowledger, DeliveryTag: 1, RoutingKey: routingKey, Body: body}, acknowledger
}

func testConsumer() *Consumer {
	return &Consumer{processingTimeout: time.Second, logger: slog.New(slog.DiscardHandler)}
}

func TestConsumerAcknowledgesAfterSuccessfulHandling(t *testing.T) {
	message := validPlotMessage(domain.PlotCreated)
	delivery, acknowledger := testDelivery(encodeMessage(t, message), message.EventType)
	handler := &eventHandlerStub{}

	if err := testConsumer().handleDelivery(context.Background(), delivery, handler); err != nil {
		t.Fatalf("handleDelivery() error = %v", err)
	}

	if !acknowledger.acked || acknowledger.rejected || handler.event.ID != message.EventID {
		t.Fatalf("ack=%v rejected=%v handled=%#v", acknowledger.acked, acknowledger.rejected, handler.event)
	}
}

func TestConsumerDeadLettersInvalidMessage(t *testing.T) {
	delivery, acknowledger := testDelivery([]byte(`{"broken":true}`), string(domain.PlotCreated))

	if err := testConsumer().handleDelivery(context.Background(), delivery, &eventHandlerStub{}); err != nil {
		t.Fatalf("handleDelivery() error = %v", err)
	}

	if !acknowledger.deadLettered() {
		t.Fatalf("acknowledger = %#v, want reject without requeue", acknowledger)
	}
}

func TestConsumerDeadLettersProcessingFailure(t *testing.T) {
	message := validPlotMessage(domain.PlotUpdated)
	delivery, acknowledger := testDelivery(encodeMessage(t, message), message.EventType)
	handler := &eventHandlerStub{err: errors.New("database unavailable")}

	if err := testConsumer().handleDelivery(context.Background(), delivery, handler); err != nil {
		t.Fatalf("handleDelivery() error = %v", err)
	}

	if !acknowledger.deadLettered() {
		t.Fatalf("acknowledger = %#v, want reject without requeue", acknowledger)
	}
}
