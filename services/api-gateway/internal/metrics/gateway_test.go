package metrics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

const testService = "api-gateway"

func TestRequestDurationHasOnlyLowCardinalityLabels(t *testing.T) {
	gateway := NewGateway(testService)
	gateway.ObserveRequest("/api/v1/plots/{id}", http.MethodGet, http.StatusOK, true, 20*time.Millisecond)

	families, err := gateway.registry.Gather()
	if err != nil {
		t.Fatal(err)
	}

	for _, family := range families {
		if family.GetName() != "http_server_request_duration_seconds" {
			continue
		}

		labels := make([]string, 0)
		for _, label := range family.GetMetric()[0].GetLabel() {
			labels = append(labels, label.GetName())
		}

		slices.Sort(labels)

		want := []string{labelMethod, labelRoute, labelService, labelStatusClass}
		if !slices.Equal(labels, want) {
			t.Fatalf("labels = %v, want %v", labels, want)
		}

		return
	}

	t.Fatal("http_server_request_duration_seconds is not registered")
}

func TestRequestsAreCountedByStatusClass(t *testing.T) {
	gateway := NewGateway(testService)

	for _, status := range []int{http.StatusOK, http.StatusCreated, http.StatusBadGateway} {
		gateway.ObserveRequest("/api/v1/plots", http.MethodPost, status, true, time.Millisecond)
	}

	for class, want := range map[string]int{"2xx": 2, "5xx": 1} {
		if got := sampleCount(t, gateway, class); got != want {
			t.Fatalf("observations of %s = %d, want %d", class, got, want)
		}
	}

	if count := testutil.CollectAndCount(gateway.rejectedRequests); count != 0 {
		t.Fatalf("rejected series = %d, want 0", count)
	}
}

func TestRejectionsAreCountedByStatusAndOrigin(t *testing.T) {
	gateway := NewGateway(testService)
	gateway.ObserveRequest("/api/v1/plots", http.MethodGet, http.StatusUnauthorized, false, time.Millisecond)
	gateway.ObserveRequest("/api/v1/plots", http.MethodGet, http.StatusUnauthorized, false, time.Millisecond)
	gateway.ObserveRequest(
		"/api/v1/plots/preview",
		http.MethodPost,
		http.StatusTooManyRequests,
		false,
		time.Millisecond,
	)
	gateway.ObserveRequest("/api/v1/plots/{id}", http.MethodGet, http.StatusForbidden, true, time.Millisecond)
	gateway.ObserveRequest("/api/v1/plots/{id}", http.MethodGet, http.StatusNotFound, true, time.Millisecond)

	want := `
		# HELP gateway_rejected_requests_total Responses with status 401, 403 or 429, by who produced them.
		# TYPE gateway_rejected_requests_total counter
		gateway_rejected_requests_total{origin="gateway",service="api-gateway",status="401"} 2
		gateway_rejected_requests_total{origin="gateway",service="api-gateway",status="429"} 1
		gateway_rejected_requests_total{origin="upstream",service="api-gateway",status="403"} 1
	`
	if err := testutil.CollectAndCompare(gateway.rejectedRequests, strings.NewReader(want)); err != nil {
		t.Fatal(err)
	}
}

func TestUpstreamOutcomes(t *testing.T) {
	cases := []struct {
		status int
		err    error
		want   string
	}{
		{status: http.StatusOK, want: "2xx"},
		{status: http.StatusServiceUnavailable, want: "5xx"},
		{err: context.DeadlineExceeded, want: outcomeTimeout},
		{err: fmt.Errorf("dial: %w", context.DeadlineExceeded), want: outcomeTimeout},
		{err: errors.New("connection refused"), want: outcomeError},
	}
	for _, testCase := range cases {
		if got := upstreamOutcome(testCase.status, testCase.err); got != testCase.want {
			t.Fatalf("upstreamOutcome(%d, %v) = %q, want %q", testCase.status, testCase.err, got, testCase.want)
		}
	}
}

func TestUpstreamDurationIsLabelledByService(t *testing.T) {
	gateway := NewGateway(testService)
	gateway.ObserveUpstream("report-service", http.StatusOK, nil, 30*time.Millisecond)
	gateway.ObserveUpstream("report-service", 0, context.DeadlineExceeded, 5*time.Second)

	if count := testutil.CollectAndCount(gateway.upstreamDuration); count != 2 {
		t.Fatalf("series = %d, want 2", count)
	}
}

func TestHandlerExposesProcessAndGatewayMetrics(t *testing.T) {
	gateway := NewGateway(testService)
	gateway.ObserveRequest("/health", http.MethodGet, http.StatusOK, false, time.Millisecond)

	response := httptest.NewRecorder()
	gateway.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody))

	body := response.Body.String()
	for _, name := range []string{"http_server_request_duration_seconds_bucket", "go_goroutines", "process_cpu_seconds_total"} {
		if !strings.Contains(body, name) {
			t.Fatalf("metrics output has no %s", name)
		}
	}
}

func sampleCount(t *testing.T, gateway *Gateway, class string) int {
	t.Helper()

	families, err := gateway.registry.Gather()
	if err != nil {
		t.Fatal(err)
	}

	for _, family := range families {
		if family.GetName() != "http_server_request_duration_seconds" {
			continue
		}

		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == labelStatusClass && label.GetValue() == class {
					return int(metric.GetHistogram().GetSampleCount())
				}
			}
		}
	}

	return 0
}
