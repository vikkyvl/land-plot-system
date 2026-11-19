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

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

const testService = "report-service"

type statsStub struct {
	backlog domain.OutboxBacklog
	counts  domain.ReportCounts
	err     error
}

func (stub statsStub) Backlog(context.Context) (domain.OutboxBacklog, error) {
	return stub.backlog, stub.err
}

func (stub statsStub) Counts(context.Context) (domain.ReportCounts, error) {
	return stub.counts, stub.err
}

func newTestMetrics(stats statsStub) *Service {
	return New(testService, Stats{Backlog: stats, Reports: stats, Timeout: time.Second})
}

func TestReportCounters(t *testing.T) {
	service := newTestMetrics(statsStub{})
	service.ReportCreated(domain.DataAvailable, domain.DataAvailable)
	service.ReportCreated(domain.DataUnavailable, domain.DataUnavailable)
	service.ReportCreated(domain.DataAvailable, domain.DataUnavailable)
	service.ReportDownloaded(domain.ExtensionPDF)
	service.ReportDownloaded(domain.ExtensionPDF)
	service.ReportDownloaded(domain.ExtensionGeoJSON)

	want := `
		# HELP reports_created_total Reports created.
		# TYPE reports_created_total counter
		reports_created_total{service="report-service"} 3
		# HELP report_blocks_unavailable_total Reports created while the terrain or climate source did not answer.
		# TYPE report_blocks_unavailable_total counter
		report_blocks_unavailable_total{block="climate",service="report-service"} 2
		report_blocks_unavailable_total{block="terrain",service="report-service"} 1
		# HELP report_downloads_total Report files built for download, by format.
		# TYPE report_downloads_total counter
		report_downloads_total{format="geojson",service="report-service"} 1
		report_downloads_total{format="pdf",service="report-service"} 2
	`

	err := testutil.GatherAndCompare(
		service.registry,
		strings.NewReader(want),
		"reports_created_total", "report_blocks_unavailable_total", "report_downloads_total",
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestOutgoingCallsAreMeasuredByTargetAndOutcome(t *testing.T) {
	service := newTestMetrics(statsStub{})

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/slow":
			time.Sleep(200 * time.Millisecond)
		case "/broken":
			response.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer server.Close()

	client := &http.Client{
		Timeout:   50 * time.Millisecond,
		Transport: service.Transport(TargetOpenMeteoArchive, http.DefaultTransport),
	}

	for _, path := range []string{"/ok", "/broken", "/slow"} {
		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+path, http.NoBody)
		if response, err := client.Do(request); err == nil {
			_ = response.Body.Close()
		}
	}

	unreachable := &http.Client{Transport: service.Transport(TargetLandPlot, http.DefaultTransport)}
	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:1/", http.NoBody)

	if response, err := unreachable.Do(request); err == nil {
		_ = response.Body.Close()
	}

	got := map[string]uint64{}

	families, err := service.registry.Gather()
	if err != nil {
		t.Fatal(err)
	}

	for _, family := range families {
		if family.GetName() != "http_client_request_duration_seconds" {
			continue
		}

		for _, metric := range family.GetMetric() {
			labels := map[string]string{}
			for _, label := range metric.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}

			got[labels[labelTarget]+" "+labels[labelOutcome]] = metric.GetHistogram().GetSampleCount()
		}
	}

	for _, key := range []string{
		TargetOpenMeteoArchive + " 2xx",
		TargetOpenMeteoArchive + " 5xx",
		TargetOpenMeteoArchive + " " + outcomeTimeout,
		TargetLandPlot + " " + outcomeError,
	} {
		if got[key] != 1 {
			t.Fatalf("series = %v, want one call for %q", got, key)
		}
	}
}

func TestDatabaseStatsAreReadOnScrape(t *testing.T) {
	service := newTestMetrics(statsStub{
		backlog: domain.OutboxBacklog{Pending: 2, OldestAge: 3 * time.Second},
		counts:  domain.ReportCounts{ActivePlots: 7, DeletedPlots: 2},
	})

	want := `
		# HELP database_stats_up 1 when the outbox and report statistics could be read on this scrape.
		# TYPE database_stats_up gauge
		database_stats_up{service="report-service"} 1
		# HELP outbox_oldest_age_seconds Age of the oldest pending outbox row, 0 when none is pending.
		# TYPE outbox_oldest_age_seconds gauge
		outbox_oldest_age_seconds{service="report-service"} 3
		# HELP reports_stored Reports in report_db by whether their plot still exists.
		# TYPE reports_stored gauge
		reports_stored{plot="active",service="report-service"} 7
		reports_stored{plot="deleted",service="report-service"} 2
	`

	err := testutil.GatherAndCompare(
		service.registry, strings.NewReader(want), "database_stats_up", "outbox_oldest_age_seconds", "reports_stored",
	)
	if err != nil {
		t.Fatal(err)
	}

	failing := newTestMetrics(statsStub{err: errors.New("database is down")})
	down := `
		# HELP database_stats_up 1 when the outbox and report statistics could be read on this scrape.
		# TYPE database_stats_up gauge
		database_stats_up{service="report-service"} 0
	`

	if err := testutil.GatherAndCompare(
		failing.registry, strings.NewReader(down), "database_stats_up", "outbox_pending", "reports_stored",
	); err != nil {
		t.Fatal(err)
	}
}

func TestDeliveriesAndMessageAge(t *testing.T) {
	service := newTestMetrics(statsStub{})
	now := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	service.DeliveryProcessed("report.plot-events", "ack", now.Add(-2*time.Second))

	if count := testutil.CollectAndCount(service.messageAge); count != 1 {
		t.Fatalf("message age series = %d", count)
	}

	if value := testutil.ToFloat64(service.deliveries.WithLabelValues("report.plot-events", "ack")); value != 1 {
		t.Fatalf("processed = %v", value)
	}
}

func TestRequestsAreLabelledWithTheMuxPattern(t *testing.T) {
	service := newTestMetrics(statsStub{})
	router := http.NewServeMux()
	router.HandleFunc("GET /api/v1/reports/{id}/pdf", func(http.ResponseWriter, *http.Request) {})

	service.ObserveRequests(router).ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/api/v1/reports/5b1e/pdf", http.NoBody),
	)

	families, err := service.registry.Gather()
	if err != nil {
		t.Fatal(err)
	}

	for _, family := range families {
		if family.GetName() != "http_server_request_duration_seconds" {
			continue
		}

		labels := map[string]string{}
		for _, label := range family.GetMetric()[0].GetLabel() {
			labels[label.GetName()] = label.GetValue()
		}

		if len(family.GetMetric()) != 1 || labels[labelRoute] != "/api/v1/reports/{id}/pdf" ||
			labels[labelStatusClass] != "2xx" || labels[labelService] != testService {
			t.Fatalf("series = %v", family.GetMetric())
		}

		return
	}

	t.Fatal("no request was measured")
}
