package rabbitmq

import (
	"context"
	"log/slog"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

const (
	testSagaID       = "6c9f1f0e-3a55-4b44-9f43-2f2c1c3b7a10"
	testReplyEventID = "0f0b6b43-7c55-4b8e-9d4c-1d3f8b0e2a11"
)

type acknowledgerStub struct {
	acknowledged bool
	rejected     bool
	requeued     bool
}

func (stub *acknowledgerStub) Ack(uint64, bool) error {
	stub.acknowledged = true
	return nil
}

func (stub *acknowledgerStub) Nack(_ uint64, _, requeue bool) error {
	stub.requeued = requeue
	return nil
}

func (stub *acknowledgerStub) Reject(uint64, bool) error {
	stub.rejected = true
	return nil
}

type sagaHandlerStub struct {
	confirmed   []string
	compensated []string
	reason      domain.CompensationReason
	err         error
}

func (stub *sagaHandlerStub) Confirm(_ context.Context, sagaID string) error {
	stub.confirmed = append(stub.confirmed, sagaID)
	return stub.err
}

func (stub *sagaHandlerStub) Compensate(_ context.Context, sagaID string, reason domain.CompensationReason) error {
	stub.compensated = append(stub.compensated, sagaID)
	stub.reason = reason

	return stub.err
}

func testConsumer() *Consumer {
	return NewConsumer(ConsumerConfig{ProcessingTimeout: time.Second}, slog.New(slog.DiscardHandler))
}

func summaryAppliedBody(appliedEventID string) []byte {
	return []byte(`{"schema_version":1,"event_id":"` + testReplyEventID +
		`","event_type":"plot.summary.applied","occurred_at":"2026-11-08T10:00:00Z","applied_event_id":"` +
		appliedEventID + `","plot_id":"` + testSagaID + `","plot_version":1}`)
}

func handle(t *testing.T, delivery amqp.Delivery, handler SagaHandler) *acknowledgerStub {
	t.Helper()

	acknowledger := &acknowledgerStub{}
	delivery.Acknowledger = acknowledger

	if err := testConsumer().handleDelivery(context.Background(), delivery, handler); err != nil {
		t.Fatalf("handleDelivery() error = %v", err)
	}

	return acknowledger
}

func TestConsumerConfirmsSagaFromSummaryApplied(t *testing.T) {
	handler := &sagaHandlerStub{}

	acknowledger := handle(t, amqp.Delivery{
		Exchange:   exchangeName,
		RoutingKey: summaryAppliedEvent,
		Body:       summaryAppliedBody(testSagaID),
	}, handler)

	if !acknowledger.acknowledged || len(handler.confirmed) != 1 || handler.confirmed[0] != testSagaID {
		t.Fatalf("acknowledged = %v, confirmed = %v", acknowledger.acknowledged, handler.confirmed)
	}
}

func TestConsumerCompensatesSagaFromDeadLetter(t *testing.T) {
	handler := &sagaHandlerStub{}

	acknowledger := handle(t, amqp.Delivery{
		Exchange:   deadExchangeName,
		RoutingKey: string(domain.PlotCreated),
		MessageId:  testSagaID,
	}, handler)

	if !acknowledger.acknowledged || len(handler.compensated) != 1 || handler.reason != domain.ReasonDeadLetter {
		t.Fatalf("acknowledged = %v, compensated = %v, reason = %s",
			acknowledger.acknowledged, handler.compensated, handler.reason)
	}
}

func TestConsumerRejectsInvalidSagaMessage(t *testing.T) {
	handler := &sagaHandlerStub{}

	acknowledger := handle(t, amqp.Delivery{
		Exchange:   exchangeName,
		RoutingKey: summaryAppliedEvent,
		Body:       summaryAppliedBody("not-a-uuid"),
	}, handler)

	if !acknowledger.rejected || len(handler.confirmed) != 0 {
		t.Fatalf("rejected = %v, confirmed = %v", acknowledger.rejected, handler.confirmed)
	}
}

func TestConsumerRequeuesWhenHandlerFails(t *testing.T) {
	handler := &sagaHandlerStub{err: errBrokerRejectedMessage}

	acknowledger := handle(t, amqp.Delivery{
		Exchange:  deadExchangeName,
		MessageId: testSagaID,
	}, handler)

	if !acknowledger.requeued || acknowledger.acknowledged {
		t.Fatalf("requeued = %v, acknowledged = %v", acknowledger.requeued, acknowledger.acknowledged)
	}
}
