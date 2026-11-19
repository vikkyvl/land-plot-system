package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

const (
	instrumentationName = "github.com/vleshchenko/land-plot-system/services/report-service/internal/messaging/rabbitmq"
	messagingSystem     = "rabbitmq"
	publishOperation    = "publish"
	processOperation    = "process"
	spanNameSeparator   = " "

	attributeSystem      = "messaging.system"
	attributeOperation   = "messaging.operation.type"
	attributeDestination = "messaging.destination.name"
	attributeRoutingKey  = "messaging.rabbitmq.destination.routing_key"
	attributeMessageID   = "messaging.message.id"
	attributeOutcome     = "messaging.outcome"
)

type headerCarrier amqp.Table

func (carrier headerCarrier) Get(key string) string {
	value, _ := carrier[key].(string)

	return value
}

func (carrier headerCarrier) Set(key, value string) {
	carrier[key] = value
}

func (carrier headerCarrier) Keys() []string {
	keys := make([]string, 0, len(carrier))
	for key := range carrier {
		keys = append(keys, key)
	}

	return keys
}

func startPublishSpan(
	publishContext context.Context,
	outboxMessage domain.OutboxMessage,
	headers amqp.Table,
) trace.Span {
	parent := otel.GetTextMapPropagator().Extract(
		publishContext,
		propagation.MapCarrier(outboxMessage.TraceContext),
	)
	spanContext, span := otel.Tracer(instrumentationName).Start(
		parent,
		publishOperation+spanNameSeparator+outboxMessage.RoutingKey,
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String(attributeSystem, messagingSystem),
			attribute.String(attributeOperation, publishOperation),
			attribute.String(attributeDestination, exchangeName),
			attribute.String(attributeRoutingKey, outboxMessage.RoutingKey),
			attribute.String(attributeMessageID, outboxMessage.ID),
		),
	)
	otel.GetTextMapPropagator().Inject(spanContext, headerCarrier(headers))

	return span
}

func startProcessSpan(runContext context.Context, delivery amqp.Delivery, queue string) (context.Context, trace.Span) {
	published := otel.GetTextMapPropagator().Extract(runContext, headerCarrier(delivery.Headers))

	return otel.Tracer(instrumentationName).Start(
		published,
		processOperation+spanNameSeparator+queue,
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithLinks(trace.LinkFromContext(published)),
		trace.WithAttributes(
			attribute.String(attributeSystem, messagingSystem),
			attribute.String(attributeOperation, processOperation),
			attribute.String(attributeDestination, queue),
			attribute.String(attributeRoutingKey, delivery.RoutingKey),
			attribute.String(attributeMessageID, delivery.MessageId),
		),
	)
}

func finishSpan(span trace.Span, outcome string, err error) {
	span.SetAttributes(attribute.String(attributeOutcome, outcome))

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	span.End()
}
