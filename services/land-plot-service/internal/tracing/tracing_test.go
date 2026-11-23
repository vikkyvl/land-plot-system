package tracing

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

const (
	incomingTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	incomingSpanID  = "00f067aa0ba902b7"
	traceparent     = "00-" + incomingTraceID + "-" + incomingSpanID + "-01"
)

func recordSpans(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()

	recorder := tracetest.NewSpanRecorder()
	previousProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()

	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})

	return recorder
}

func patternRoute(request *http.Request) string {
	_, path, _ := strings.Cut(request.Pattern, " ")

	return path
}

func TestServerContinuesTheIncomingTraceAndNamesTheSpanByRoute(t *testing.T) {
	recorder := recordSpans(t)
	router := http.NewServeMux()
	router.HandleFunc("GET /api/v1/items/{id}", func(http.ResponseWriter, *http.Request) {})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/items/42", http.NoBody)
	request.Header.Set("traceparent", traceparent)
	Server(NameByRoute(patternRoute, router)).ServeHTTP(httptest.NewRecorder(), request)

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("spans = %d, want 1", len(spans))
	}

	span := spans[0]
	if span.SpanContext().TraceID().String() != incomingTraceID || span.Parent().SpanID().String() != incomingSpanID {
		t.Fatalf("trace = %s, parent = %s", span.SpanContext().TraceID(), span.Parent().SpanID())
	}

	if span.Name() != "GET /api/v1/items/{id}" {
		t.Fatalf("span name = %q", span.Name())
	}
}

func TestServerStartsATraceWithoutTraceparentAndSkipsHealthChecks(t *testing.T) {
	recorder := recordSpans(t)
	handler := Server(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, healthPath, http.NoBody))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/items", http.NoBody))

	spans := recorder.Ended()
	if len(spans) != 1 || !spans[0].SpanContext().IsValid() || spans[0].Parent().IsValid() {
		t.Fatalf("spans = %v", spans)
	}
}

func TestTransportsCreateClientSpansAndPropagateOnlyWhenAllowed(t *testing.T) {
	recorder := recordSpans(t)
	received := make(chan string, 2)

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		received <- request.Header.Get("traceparent")
	}))
	defer server.Close()

	for _, transport := range []http.RoundTripper{
		Transport(http.DefaultTransport),
		TransportWithoutPropagation(http.DefaultTransport),
	} {
		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)

		response, err := (&http.Client{Transport: transport}).Do(request)
		if err != nil {
			t.Fatal(err)
		}

		_ = response.Body.Close()
	}

	if internal, external := <-received, <-received; internal == "" || external != "" {
		t.Fatalf("traceparent to the service = %q, to the external API = %q", internal, external)
	}

	if spans := recorder.Ended(); len(spans) != 2 {
		t.Fatalf("client spans = %d, want 2", len(spans))
	}
}

func TestStartWithoutEndpointKeepsTracingOff(t *testing.T) {
	stop := Start(context.Background(), Config{ServiceName: "test"}, slog.New(slog.DiscardHandler))
	stop(context.Background())

	traceID, _ := trace.TraceIDFromHex(incomingTraceID)
	spanID, _ := trace.SpanIDFromHex(incomingSpanID)
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled,
	})
	carrier := propagation.MapCarrier{}

	otel.GetTextMapPropagator().Inject(trace.ContextWithSpanContext(context.Background(), spanContext), carrier)

	if carrier.Get("traceparent") != traceparent {
		t.Fatalf("traceparent = %q, want %q", carrier.Get("traceparent"), traceparent)
	}
}
