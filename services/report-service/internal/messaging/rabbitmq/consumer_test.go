package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/repository/postgres"
)

type acknowledgerStub struct {
	acked    bool
	rejected bool
	requeue  bool
	nacked   bool
}

func (stub *acknowledgerStub) Ack(uint64, bool) error {
	stub.acked = true
	return nil
}

func (stub *acknowledgerStub) Nack(_ uint64, _, requeue bool) error {
	stub.nacked = true
	stub.requeue = requeue

	return nil
}

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

type deliveryObservation struct {
	queue     string
	outcome   string
	createdAt time.Time
}

type deliveryObserverStub struct {
	observations []deliveryObservation
}

func (stub *deliveryObserverStub) DeliveryProcessed(queue, outcome string, createdAt time.Time) {
	stub.observations = append(
		stub.observations,
		deliveryObservation{queue: queue, outcome: outcome, createdAt: createdAt},
	)
}

func testConsumer() *Consumer {
	return observedConsumer(&deliveryObserverStub{})
}

func observedConsumer(observer DeliveryObserver) *Consumer {
	return NewConsumer(ConsumerConfig{ProcessingTimeout: time.Second}, observer, slog.New(slog.DiscardHandler))
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

func TestConsumerDeadLettersRejectedEvent(t *testing.T) {
	message := validPlotMessage(domain.PlotCreated)
	delivery, acknowledger := testDelivery(encodeMessage(t, message), message.EventType)
	handler := &eventHandlerStub{err: fmt.Errorf("apply plot event: %w", postgres.ErrSimulatedWriteFailure)}

	if err := testConsumer().handleDelivery(context.Background(), delivery, handler); err != nil {
		t.Fatalf("handleDelivery() error = %v", err)
	}

	if !acknowledger.deadLettered() || acknowledger.nacked {
		t.Fatalf("acknowledger = %#v, want reject without requeue", acknowledger)
	}
}

func TestConsumerRequeuesTemporaryFailure(t *testing.T) {
	for name, cause := range map[string]error{
		"deadline exceeded": context.DeadlineExceeded,
		"database error":    errors.New("database unavailable"),
	} {
		t.Run(name, func(t *testing.T) {
			message := validPlotMessage(domain.PlotUpdated)
			delivery, acknowledger := testDelivery(encodeMessage(t, message), message.EventType)

			err := testConsumer().handleDelivery(context.Background(), delivery, &eventHandlerStub{err: cause})
			if err != nil {
				t.Fatalf("handleDelivery() error = %v", err)
			}

			if !acknowledger.nacked || !acknowledger.requeue || acknowledger.rejected || acknowledger.acked {
				t.Fatalf("acknowledger = %#v, want nack with requeue", acknowledger)
			}
		})
	}
}

func TestConsumerAcknowledgesCancellation(t *testing.T) {
	message := validPlotMessage(domain.PlotCancelled)
	delivery, acknowledger := testDelivery(encodeMessage(t, message), message.EventType)
	handler := &eventHandlerStub{}

	if err := testConsumer().handleDelivery(context.Background(), delivery, handler); err != nil {
		t.Fatalf("handleDelivery() error = %v", err)
	}

	if !acknowledger.acked || handler.event.Type != domain.PlotCancelled {
		t.Fatalf("ack=%v handled=%#v", acknowledger.acked, handler.event)
	}
}

func TestConsumerObservesEveryOutcomeWithTheEventAge(t *testing.T) {
	created := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
	message := validPlotMessage(domain.PlotCreated)
	cases := []struct {
		body    []byte
		handler *eventHandlerStub
		want    string
	}{
		{body: encodeMessage(t, message), handler: &eventHandlerStub{}, want: OutcomeAcknowledged},
		{body: []byte(`{"broken":true}`), handler: &eventHandlerStub{}, want: OutcomeRejected},
		{
			body:    encodeMessage(t, message),
			handler: &eventHandlerStub{err: fmt.Errorf("apply: %w", postgres.ErrSimulatedWriteFailure)},
			want:    OutcomeRejected,
		},
		{
			body:    encodeMessage(t, message),
			handler: &eventHandlerStub{err: errors.New("database down")},
			want:    OutcomeRequeued,
		},
	}

	for _, testCase := range cases {
		observer := &deliveryObserverStub{}
		delivery, _ := testDelivery(testCase.body, message.EventType)
		delivery.Headers = amqp.Table{HeaderCreatedAt: created.UnixMilli()}

		if err := observedConsumer(
			observer,
		).handleDelivery(context.Background(), delivery, testCase.handler); err != nil {
			t.Fatalf("handleDelivery() error = %v", err)
		}

		if len(observer.observations) != 1 || observer.observations[0].queue != queueName ||
			observer.observations[0].outcome != testCase.want || !observer.observations[0].createdAt.Equal(created) {
			t.Fatalf("observations = %+v, want outcome %s", observer.observations, testCase.want)
		}
	}
}
