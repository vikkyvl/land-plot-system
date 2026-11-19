package respond

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/logging"
)

func TestInternalErrorLogKeepsRequestIDAndNoSecrets(t *testing.T) {
	var output bytes.Buffer

	token := "eyJ" + strings.Repeat("x", 3) + ".payload.signature"
	password := "pass" + "word-123"
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/anything",
		strings.NewReader(`{"password":"`+password+`"}`),
	)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Cookie", "session="+token)
	request = request.WithContext(logging.WithRequestID(request.Context(), "req-9"))

	NewWriter(logging.NewLogger(&output, "land-plot-service", slog.LevelInfo)).
		Error(httptest.NewRecorder(), request, errors.New("database is down"))

	logged := output.String()
	if !strings.Contains(logged, `"request_id":"req-9"`) || !strings.Contains(logged, `"level":"ERROR"`) {
		t.Fatalf("log = %s", logged)
	}

	for _, secret := range []string{token, password, "Authorization", "Cookie"} {
		if strings.Contains(logged, secret) {
			t.Fatalf("log contains %q: %s", secret, logged)
		}
	}
}

func TestErrorBodyCarriesTheTraceID(t *testing.T) {
	traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: spanID})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/anything", http.NoBody)
	request.Header.Set(RequestIDHeader, "req-9")
	request = request.WithContext(trace.ContextWithSpanContext(request.Context(), spanContext))
	response := httptest.NewRecorder()

	NewWriter(slog.New(slog.DiscardHandler)).Error(response, request, errors.New("database is down"))

	var body ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	if body.TraceID != traceID.String() || body.RequestID != "req-9" {
		t.Fatalf("body = %+v", body)
	}
}
