package bootstrap

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/handler"
	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/proxy"
	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/tracing"
)

const incomingTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"

type allowAll struct{}

func (allowAll) Allow(string) (bool, time.Duration) { return true, 0 }

type noObserver struct{}

func (noObserver) ObserveRequest(string, string, int, bool, time.Duration) {}

type rejectAll struct{}

func (rejectAll) Verify(string) (proxy.Principal, error) {
	return proxy.Principal{}, proxy.ErrUnauthorized
}

func tracedGateway(t *testing.T, upstream http.Handler) (http.Handler, *tracetest.SpanRecorder) {
	t.Helper()

	recorder := tracetest.NewSpanRecorder()
	previousProvider, previousPropagator := otel.GetTracerProvider(), otel.GetTextMapPropagator()

	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})

	router := handler.NewRouter(
		slog.New(slog.DiscardHandler), "api-gateway", upstream, rejectAll{}, allowAll{}, nil, noObserver{},
	)

	return tracing.Server(tracing.NameByRoute(gatewayRoute, router)), recorder
}

func TestGatewayContinuesTheTraceAndReturnsItsID(t *testing.T) {
	gateway, recorder := tracedGateway(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/plots/preview", http.NoBody)
	request.Header.Set("traceparent", "00-"+incomingTraceID+"-00f067aa0ba902b7-01")

	response := httptest.NewRecorder()

	gateway.ServeHTTP(response, request)

	if response.Header().Get(proxy.HeaderTraceID) != incomingTraceID {
		t.Fatalf("%s = %q, want %s", proxy.HeaderTraceID, response.Header().Get(proxy.HeaderTraceID), incomingTraceID)
	}

	spans := recorder.Ended()
	if len(spans) != 1 || spans[0].Name() != "POST /api/v1/plots/preview" {
		t.Fatalf("spans = %v", spans)
	}
}

func TestGatewayStartsATraceAndPutsItIntoTheErrorBody(t *testing.T) {
	gateway, _ := tracedGateway(
		t,
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("upstream called") }),
	)
	response := httptest.NewRecorder()

	gateway.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/plots", http.NoBody))

	var body struct {
		RequestID string `json:"request_id"`
		TraceID   string `json:"trace_id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	traceID := response.Header().Get(proxy.HeaderTraceID)
	if response.Code != http.StatusUnauthorized || traceID == "" || body.TraceID != traceID || body.RequestID == "" {
		t.Fatalf("status = %d, header = %q, body = %+v", response.Code, traceID, body)
	}
}
