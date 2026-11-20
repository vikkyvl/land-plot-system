package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

const testService = "identity-service"

func newTestLogger(output *bytes.Buffer) *slog.Logger {
	return NewLogger(output, testService, slog.LevelInfo)
}

func logLines(t *testing.T, output *bytes.Buffer) []map[string]any {
	t.Helper()

	lines := make([]map[string]any, 0)

	for line := range strings.SplitSeq(strings.TrimSpace(output.String()), "\n") {
		if line == "" {
			continue
		}

		var fields map[string]any
		if err := json.Unmarshal([]byte(line), &fields); err != nil {
			t.Fatalf("log line %q is not JSON: %v", line, err)
		}

		lines = append(lines, fields)
	}

	return lines
}

func TestLoggerAddsServiceAndRequestID(t *testing.T) {
	var output bytes.Buffer

	newTestLogger(&output).InfoContext(WithRequestID(context.Background(), "req-1"), "event")

	line := logLines(t, &output)[0]
	if line[KeyService] != testService || line[KeyRequestID] != "req-1" || line["msg"] != "event" {
		t.Fatalf("line = %v", line)
	}

	if _, found := line[KeyTraceID]; found {
		t.Fatalf("trace_id written without a span: %v", line)
	}
}

func TestLoggerAddsTraceAndSpanFromContext(t *testing.T) {
	var output bytes.Buffer

	traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: spanID})

	newTestLogger(&output).InfoContext(trace.ContextWithSpanContext(context.Background(), spanContext), "event")

	line := logLines(t, &output)[0]
	if line[KeyTraceID] != traceID.String() || line[KeySpanID] != spanID.String() {
		t.Fatalf("line = %v", line)
	}
}

func TestAccessLogWritesOneLinePerRequestWithLevelByStatus(t *testing.T) {
	cases := []struct {
		status int
		level  string
	}{
		{status: http.StatusOK, level: "INFO"},
		{status: http.StatusNotFound, level: "WARN"},
		{status: http.StatusBadGateway, level: "ERROR"},
	}

	for _, testCase := range cases {
		var output bytes.Buffer

		router := http.NewServeMux()
		router.HandleFunc("GET /api/v1/items/{id}", func(response http.ResponseWriter, _ *http.Request) {
			response.WriteHeader(testCase.status)
		})

		request := httptest.NewRequest(http.MethodGet, "/api/v1/items/42", http.NoBody)
		request.Header.Set(HeaderRequestID, "req-7")
		AccessLog(newTestLogger(&output), PatternRoute, router).ServeHTTP(httptest.NewRecorder(), request)

		lines := logLines(t, &output)
		if len(lines) != 1 {
			t.Fatalf("lines = %v, want one", lines)
		}

		line := lines[0]
		if line["level"] != testCase.level || line[KeyRoute] != "/api/v1/items/{id}" ||
			line[KeyStatus] != float64(testCase.status) || line[KeyRequestID] != "req-7" ||
			line[KeyMethod] != http.MethodGet || line[KeyService] != testService {
			t.Fatalf("line = %v", line)
		}

		if _, found := line[KeyDurationMS]; !found {
			t.Fatalf("line has no %s: %v", KeyDurationMS, line)
		}
	}
}

func TestAccessLogKeepsHealthChecksOutOfInfo(t *testing.T) {
	var output bytes.Buffer

	router := http.NewServeMux()
	router.HandleFunc("GET /health", func(http.ResponseWriter, *http.Request) {})
	AccessLog(newTestLogger(&output), PatternRoute, router).
		ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, HealthRoute, http.NoBody))

	if output.Len() != 0 {
		t.Fatalf("health check logged at info: %s", output.String())
	}
}

func TestAccessLogGivesEveryRequestAnIDForInnerLogs(t *testing.T) {
	var output bytes.Buffer

	logger := newTestLogger(&output)
	inner := http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		logger.InfoContext(request.Context(), "inner")
	})

	AccessLog(logger, PatternRoute, inner).
		ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/anything", http.NoBody))

	lines := logLines(t, &output)
	if len(lines) != 2 || lines[0][KeyRequestID] == "" || lines[0][KeyRequestID] != lines[1][KeyRequestID] ||
		lines[1][KeyRoute] != UnmatchedRoute {
		t.Fatalf("lines = %v", lines)
	}
}

func TestAccessLogNeverWritesCredentialsOrBodies(t *testing.T) {
	var output bytes.Buffer

	secrets := []string{
		"Bearer " + strings.Repeat("t", 3) + "secret-access-token",
		"session=" + strings.Repeat("c", 3) + "cookie-value",
		"pass" + "word-in-body-123",
		"query-" + "token-456",
	}
	logger := newTestLogger(&output)
	failing := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		logger.ErrorContext(request.Context(), "request failed", KeyError, "upstream unavailable")
		response.WriteHeader(http.StatusInternalServerError)
	})

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/login?access_token="+secrets[3],
		strings.NewReader(`{"password":"`+secrets[2]+`"}`),
	)
	request.Header.Set("Authorization", secrets[0])
	request.Header.Set("Cookie", secrets[1])

	AccessLog(logger, PatternRoute, failing).ServeHTTP(httptest.NewRecorder(), request)

	logged := output.String()
	if !strings.Contains(logged, `"status":500`) {
		t.Fatalf("access line missing: %s", logged)
	}

	for _, secret := range secrets {
		if strings.Contains(logged, secret) {
			t.Fatalf("log contains %q: %s", secret, logged)
		}
	}

	for _, field := range []string{"Authorization", "Cookie", "password", "access_token"} {
		if strings.Contains(logged, field) {
			t.Fatalf("log mentions %q: %s", field, logged)
		}
	}
}
