package mcpserver

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/vleshchenko/land-plot-system/ai/internal/logging"
	"github.com/vleshchenko/land-plot-system/ai/internal/tools"
	"github.com/vleshchenko/land-plot-system/ai/internal/tracing"
)

const incomingTrace = "4bf92f3577b34da6a3ce929d0e0e4736"

type contextHeaders struct {
	base http.RoundTripper
}

func (transport contextHeaders) RoundTrip(request *http.Request) (*http.Response, error) {
	request = request.Clone(request.Context())
	request.Header.Set("Authorization", "Bearer "+testAccessToken)
	request.Header.Set(logging.HeaderRequestID, "req-42")
	request.Header.Set("traceparent", "00-"+incomingTrace+"-00f067aa0ba902b7-01")

	return transport.base.RoundTrip(request)
}

func TestToolCallsOverHTTPKeepTheRequestAndTrace(t *testing.T) {
	previousProvider, previousPropagator := otel.GetTracerProvider(), otel.GetTextMapPropagator()

	otel.SetTracerProvider(sdktrace.NewTracerProvider())
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})

	api := &fakeLandAPI{}
	logger := slog.New(slog.DiscardHandler)
	handler := tracing.Server(
		logging.AccessLog(logger, logging.PatternRoute, NewHTTPHandler(newTestServer(api, 30), testAccessToken)),
	)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)

	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:   server.URL + MCPPath,
		HTTPClient: &http.Client{Transport: contextHeaders{base: http.DefaultTransport}},
	}, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	t.Cleanup(func() { _ = session.Close() })

	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tools.ListPlots}); err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}

	if api.requestID != "req-42" || api.traceID != incomingTrace {
		t.Fatalf("tool context: request ID = %q, trace = %q", api.requestID, api.traceID)
	}
}
