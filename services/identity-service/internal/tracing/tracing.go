package tracing

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/logging"
)

const (
	tracesPath     = "/v1/traces"
	serviceNameKey = "service.name"

	logExportSpans    = "export spans"
	logTracingStopped = "stop tracing"
	logTracingOff     = "tracing is off"
)

type Config struct {
	ServiceName string
	Endpoint    string
}

func Start(startContext context.Context, config Config, logger *slog.Logger) func(context.Context) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		logger.Debug(logExportSpans, logging.KeyError, err)
	}))

	if config.Endpoint == "" {
		logger.Info(logTracingOff)
		return func(context.Context) {}
	}

	exporter, err := otlptracehttp.New(startContext, otlptracehttp.WithEndpointURL(config.Endpoint+tracesPath))
	if err != nil {
		logger.Warn(logTracingOff, logging.KeyError, err)
		return func(context.Context) {}
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewSchemaless(attribute.String(serviceNameKey, config.ServiceName))),
	)
	otel.SetTracerProvider(provider)

	return func(stopContext context.Context) {
		if err := provider.Shutdown(stopContext); err != nil {
			logger.Warn(logTracingStopped, logging.KeyError, err)
		}
	}
}
