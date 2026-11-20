package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

const instrumentationName = "github.com/vleshchenko/land-plot-system/services/report-service"

type Resumer struct{}

func Inject(source context.Context) domain.TraceContext {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(source, carrier)

	if len(carrier) == 0 {
		return nil
	}

	return domain.TraceContext(carrier)
}

func Extract(parent context.Context, traceContext domain.TraceContext) context.Context {
	return otel.GetTextMapPropagator().Extract(parent, propagation.MapCarrier(traceContext))
}

func (Resumer) Resume(
	parent context.Context,
	traceContext domain.TraceContext,
	operation string,
) (context.Context, func()) {
	resumed, span := otel.Tracer(instrumentationName).Start(Extract(parent, traceContext), operation)

	return resumed, func() { span.End() }
}
