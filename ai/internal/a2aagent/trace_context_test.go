package a2aagent

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/vleshchenko/land-plot-system/ai/internal/logging"
)

type headerRecorder struct {
	mutex       sync.Mutex
	requestIDs  []string
	traceparent []string
}

func (recorder *headerRecorder) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		recorder.mutex.Lock()
		recorder.requestIDs = append(recorder.requestIDs, request.Header.Get(logging.HeaderRequestID))
		recorder.traceparent = append(recorder.traceparent, request.Header.Get("traceparent"))
		recorder.mutex.Unlock()

		next.ServeHTTP(response, request)
	})
}

func TestDelegateCarriesTraceAndRequestIDToTheProducer(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	previousProvider, previousPropagator := otel.GetTracerProvider(), otel.GetTextMapPropagator()

	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})

	headers := &headerRecorder{}
	executor := NewExecutor(confirmedProducer(), DecideProducer, slog.New(slog.DiscardHandler))

	var server *httptest.Server

	server = httptest.NewServer(
		headers.wrap(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			NewHandler(ProducerCard(server.URL), executor).ServeHTTP(response, request)
		})),
	)
	t.Cleanup(server.Close)

	callContext, span := otel.Tracer("test").
		Start(logging.WithRequestID(context.Background(), "req-42"), "consumer task")
	output := NewProducerDelegate(server.URL, 5*time.Second, slog.New(slog.DiscardHandler)).
		Tool().Run(callContext, delegateCreateCall().Arguments)

	span.End()

	if output.IsError {
		t.Fatalf("output = %#v", output)
	}

	traceID := span.SpanContext().TraceID().String()

	headers.mutex.Lock()
	defer headers.mutex.Unlock()

	if len(headers.requestIDs) == 0 {
		t.Fatal("producer got no requests")
	}

	for index := range headers.requestIDs {
		if headers.requestIDs[index] != "req-42" || len(headers.traceparent[index]) < 36 ||
			headers.traceparent[index][3:35] != traceID {
			t.Fatalf("request %d: request ID = %q, traceparent = %q, want trace %s",
				index, headers.requestIDs[index], headers.traceparent[index], traceID)
		}
	}
}
