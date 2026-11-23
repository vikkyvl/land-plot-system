package metrics

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

const testService = "land-plot-service"

type statsStub struct {
	backlog domain.OutboxBacklog
	counts  domain.PlotCounts
	err     error
}

func (stub statsStub) Backlog(context.Context) (domain.OutboxBacklog, error) {
	return stub.backlog, stub.err
}

func (stub statsStub) Counts(context.Context) (domain.PlotCounts, error) {
	return stub.counts, stub.err
}

func newTestMetrics(stats statsStub) *Service {
	return New(testService, Stats{Backlog: stats, Plots: stats, Timeout: time.Second})
}

func TestRequestsAreLabelledWithTheMuxPattern(t *testing.T) {
	service := newTestMetrics(statsStub{})
	router := http.NewServeMux()
	router.HandleFunc("GET /api/v1/plots/{id}", func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusNotFound)
	})
	handler := service.ObserveRequests(router)

	for _, path := range []string{"/api/v1/plots/1f9b", "/api/v1/plots/7c2e", "/wp-admin/setup.php"} {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, http.NoBody))
	}

	series := map[string]int{}

	families, err := service.registry.Gather()
	if err != nil {
		t.Fatal(err)
	}

	for _, family := range families {
		if family.GetName() != "http_server_request_duration_seconds" {
			continue
		}

		for _, metric := range family.GetMetric() {
			labels := map[string]string{}
			for _, label := range metric.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}

			if len(labels) != 4 || labels[labelService] != testService {
				t.Fatalf("labels = %v", labels)
			}

			series[labels[labelRoute]+" "+labels[labelStatusClass]] = int(metric.GetHistogram().GetSampleCount())
		}
	}

	if series["/api/v1/plots/{id} 4xx"] != 2 || series[unmatchedRoute+" 4xx"] != 1 || len(series) != 2 {
		t.Fatalf("series = %v", series)
	}
}

func TestBusinessCounters(t *testing.T) {
	service := newTestMetrics(statsStub{})
	service.PlotChanged(domain.PlotCreated)
	service.PlotChanged(domain.PlotCreated)
	service.PlotChanged(domain.PlotDeleted)
	service.PlotChanged(domain.PlotCancelled)
	service.PreviewInspected(nil)
	service.PreviewInspected([]string{"self_intersection", "duplicate_point"})
	service.SagaConfirmed(domain.SagaCreate)
	service.SagaCompensated(domain.SagaCreate, domain.PlotCancelled, domain.ReasonDeadLetter)
	service.SagaCompensated(domain.SagaUpdate, domain.PlotUpdated, domain.ReasonTimeout)

	want := `
		# HELP plots_changed_total Plots created, updated or deleted.
		# TYPE plots_changed_total counter
		plots_changed_total{change="created",service="land-plot-service"} 2
		plots_changed_total{change="deleted",service="land-plot-service"} 1
		# HELP plot_previews_total Area previews by whether the polygon was valid.
		# TYPE plot_previews_total counter
		plot_previews_total{result="invalid",service="land-plot-service"} 1
		plot_previews_total{result="valid",service="land-plot-service"} 1
		# HELP plot_preview_problems_total Problems found by previews, by code.
		# TYPE plot_preview_problems_total counter
		plot_preview_problems_total{reason="duplicate_point",service="land-plot-service"} 1
		plot_preview_problems_total{reason="self_intersection",service="land-plot-service"} 1
		# HELP sagas_finished_total Sagas confirmed, cancelled (create compensated) or reverted (update compensated).
		# TYPE sagas_finished_total counter
		sagas_finished_total{kind="CREATE",reason="dlq",result="cancelled",service="land-plot-service"} 1
		sagas_finished_total{kind="CREATE",reason="none",result="confirmed",service="land-plot-service"} 1
		sagas_finished_total{kind="UPDATE",reason="timeout",result="reverted",service="land-plot-service"} 1
	`

	err := testutil.GatherAndCompare(
		service.registry,
		strings.NewReader(want),
		"plots_changed_total", "plot_previews_total", "plot_preview_problems_total", "sagas_finished_total",
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestOutboxAndDeliveryMetrics(t *testing.T) {
	service := newTestMetrics(statsStub{})
	now := time.Date(2026, 11, 19, 10, 0, 30, 0, time.UTC)
	service.now = func() time.Time { return now }

	service.OutboxPublished(3)
	service.OutboxPublishFailed(false)
	service.OutboxPublishFailed(true)
	service.DeliveryProcessed("land-plot.saga-events", "ack", now.Add(-20*time.Second))
	service.DeliveryProcessed("land-plot.saga-events", "retry", time.Time{})

	want := `
		# HELP outbox_published_total Outbox messages confirmed by the broker.
		# TYPE outbox_published_total counter
		outbox_published_total{service="land-plot-service"} 3
		# HELP outbox_publish_errors_total Failed outbox publications: retried later or abandoned after the maximum attempts.
		# TYPE outbox_publish_errors_total counter
		outbox_publish_errors_total{failure="abandoned",service="land-plot-service"} 1
		outbox_publish_errors_total{failure="retry",service="land-plot-service"} 1
		# HELP messages_processed_total Consumed messages by outcome.
		# TYPE messages_processed_total counter
		messages_processed_total{outcome="ack",queue="land-plot.saga-events",service="land-plot-service"} 1
		messages_processed_total{outcome="retry",queue="land-plot.saga-events",service="land-plot-service"} 1
	`

	err := testutil.GatherAndCompare(
		service.registry,
		strings.NewReader(want),
		"outbox_published_total", "outbox_publish_errors_total", "messages_processed_total",
	)
	if err != nil {
		t.Fatal(err)
	}

	if count := testutil.CollectAndCount(service.messageAge); count != 1 {
		t.Fatalf("message age series = %d, want 1", count)
	}
}

func TestDatabaseStatsAreReadOnScrape(t *testing.T) {
	service := newTestMetrics(statsStub{
		backlog: domain.OutboxBacklog{Pending: 4, Failed: 1, OldestAge: 45 * time.Second},
		counts:  domain.PlotCounts{Pending: 2, Confirmed: 10, Cancelled: 1, Deleted: 3},
	})

	want := `
		# HELP database_stats_up 1 when the outbox and plot statistics could be read on this scrape.
		# TYPE database_stats_up gauge
		database_stats_up{service="land-plot-service"} 1
		# HELP outbox_oldest_age_seconds Age of the oldest pending outbox row, 0 when none is pending.
		# TYPE outbox_oldest_age_seconds gauge
		outbox_oldest_age_seconds{service="land-plot-service"} 45
		# HELP outbox_pending Outbox rows not yet confirmed by the broker.
		# TYPE outbox_pending gauge
		outbox_pending{service="land-plot-service"} 4
		# HELP plots_stored Plots in plot_db by state.
		# TYPE plots_stored gauge
		plots_stored{service="land-plot-service",state="cancelled"} 1
		plots_stored{service="land-plot-service",state="confirmed"} 10
		plots_stored{service="land-plot-service",state="deleted"} 3
		plots_stored{service="land-plot-service",state="pending"} 2
	`

	err := testutil.GatherAndCompare(
		service.registry,
		strings.NewReader(want),
		"database_stats_up", "outbox_oldest_age_seconds", "outbox_pending", "plots_stored",
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDatabaseStatsFailureIsVisible(t *testing.T) {
	service := newTestMetrics(statsStub{err: errors.New("database is down")})

	want := `
		# HELP database_stats_up 1 when the outbox and plot statistics could be read on this scrape.
		# TYPE database_stats_up gauge
		database_stats_up{service="land-plot-service"} 0
	`

	err := testutil.GatherAndCompare(
		service.registry, strings.NewReader(want), "database_stats_up", "outbox_pending", "plots_stored",
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlerExposesProcessMetrics(t *testing.T) {
	response := httptest.NewRecorder()
	newTestMetrics(
		statsStub{},
	).Handler().
		ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody))

	for _, name := range []string{"process_resident_memory_bytes", "process_cpu_seconds_total", "go_goroutines"} {
		if !strings.Contains(response.Body.String(), name) {
			t.Fatalf("metrics output has no %s", name)
		}
	}
}
