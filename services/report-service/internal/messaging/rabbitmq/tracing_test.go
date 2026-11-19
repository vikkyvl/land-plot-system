package rabbitmq

import (
	"context"
	"errors"
	"strings"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

func recordSpans(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()

	recorder := tracetest.NewSpanRecorder()
	previousProvider, previousPropagator := otel.GetTracerProvider(), otel.GetTextMapPropagator()

	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})

	return recorder
}

func TestPublishSpanContinuesTheStoredTraceAndTravelsInHeaders(t *testing.T) {
	recorder := recordSpans(t)
	requestContext, request := otel.Tracer("test").Start(context.Background(), "POST /api/v1/plots")
	stored := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(requestContext, stored)
	request.End()

	headers := amqp.Table{}
	message := domain.OutboxMessage{
		ID:           "event-1",
		RoutingKey:   "plot.created",
		TraceContext: domain.TraceContext(stored),
	}
	finishSpan(startPublishSpan(context.Background(), message, headers), OutcomePublished, nil)

	publish := recorder.Ended()[1]
	if publish.Name() != "publish plot.created" || publish.SpanKind() != trace.SpanKindProducer ||
		publish.Parent().SpanID() != request.SpanContext().SpanID() ||
		publish.SpanContext().TraceID() != request.SpanContext().TraceID() {
		t.Fatalf("publish span = %s %v parent %s", publish.Name(), publish.SpanKind(), publish.Parent().SpanID())
	}

	if traceparent, _ := headers["traceparent"].(string); !strings.Contains(
		traceparent,
		publish.SpanContext().SpanID().String(),
	) {
		t.Fatalf("traceparent header = %v", headers["traceparent"])
	}
}

func TestProcessSpanIsAChildOfThePublicationWithALink(t *testing.T) {
	recorder := recordSpans(t)
	publishContext, publish := otel.Tracer("test").Start(context.Background(), "publish plot.created")
	headers := amqp.Table{}
	otel.GetTextMapPropagator().Inject(publishContext, headerCarrier(headers))
	publish.End()

	processContext, span := startProcessSpan(
		context.Background(),
		amqp.Delivery{Headers: headers, RoutingKey: "plot.created", MessageId: "event-1"},
		"test-queue",
	)
	finishSpan(span, OutcomeRequeued, errors.New("database is down"))

	process := recorder.Ended()[1]
	if process.Name() != "process test-queue" || process.SpanKind() != trace.SpanKindConsumer ||
		process.Parent().SpanID() != publish.SpanContext().SpanID() ||
		trace.SpanContextFromContext(processContext).TraceID() != publish.SpanContext().TraceID() {
		t.Fatalf("process span = %s parent %s", process.Name(), process.Parent().SpanID())
	}

	if len(process.Links()) != 1 || process.Links()[0].SpanContext.SpanID() != publish.SpanContext().SpanID() {
		t.Fatalf("links = %v", process.Links())
	}

	if process.Status().Code != codes.Error {
		t.Fatalf("status = %v", process.Status())
	}
}

func TestDeliveryWithoutTraceStartsANewOne(t *testing.T) {
	recorder := recordSpans(t)

	_, span := startProcessSpan(context.Background(), amqp.Delivery{}, "test-queue")
	finishSpan(span, OutcomeAcknowledged, nil)

	if spans := recorder.Ended(); len(spans) != 1 || spans[0].Parent().IsValid() || !spans[0].SpanContext().IsValid() {
		t.Fatalf("spans = %v", spans)
	}
}
