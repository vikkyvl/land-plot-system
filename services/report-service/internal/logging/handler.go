package logging

import (
	"context"
	"io"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

type contextHandler struct {
	next slog.Handler
}

func NewLogger(output io.Writer, serviceName string, level slog.Leveler) *slog.Logger {
	handler := slog.NewJSONHandler(output, &slog.HandlerOptions{Level: level})

	return slog.New(contextHandler{next: handler}).With(KeyService, serviceName)
}

func (handler contextHandler) Enabled(handlerContext context.Context, level slog.Level) bool {
	return handler.next.Enabled(handlerContext, level)
}

func (handler contextHandler) Handle(handlerContext context.Context, record slog.Record) error {
	if spanContext := trace.SpanContextFromContext(handlerContext); spanContext.IsValid() {
		record.AddAttrs(
			slog.String(KeyTraceID, spanContext.TraceID().String()),
			slog.String(KeySpanID, spanContext.SpanID().String()),
		)
	}

	if requestID := RequestID(handlerContext); requestID != "" {
		record.AddAttrs(slog.String(KeyRequestID, requestID))
	}

	return handler.next.Handle(handlerContext, record)
}

func (handler contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{next: handler.next.WithAttrs(attrs)}
}

func (handler contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{next: handler.next.WithGroup(name)}
}
