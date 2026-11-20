package service

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

var errRelayBroker = errors.New("broker unavailable")

type reportReaderStub struct {
	report domain.Report
	err    error
}

func (stub reportReaderStub) Get(context.Context, string, string) (domain.Report, error) {
	return stub.report, stub.err
}

type documentsStub struct{}

func (documentsStub) PDF(domain.Report, domain.Language) ([]byte, error) {
	return []byte("%PDF"), nil
}

func (documentsStub) GeoJSON(report domain.Report) ([]byte, error) {
	if len(report.Boundary) == 0 {
		return nil, domain.ErrBoundaryNotRecorded
	}

	return []byte(`{}`), nil
}

func confirmedPlot(plotID string) *plotClientStub {
	return &plotClientStub{plot: domain.PlotSnapshot{
		ID: plotID, OwnerID: testCaller.UserID, Name: "Plot", AreaM2: 10_000, PerimeterM: 400, Version: 1,
		Status: domain.PlotStatusConfirmed,
	}}
}

func TestReportServiceCountsOnlyStoredReportsWithTheirBlocks(t *testing.T) {
	metrics := &metricsSpy{}
	plotID := uuid.NewString()

	if _, err := NewReportService(
		&reportRepositoryStub{},
		confirmedPlot(plotID),
		&surroundingsStub{},
		metrics,
		testInstanceID,
	).
		Create(context.Background(), testCaller, plotID); err != nil {
		t.Fatal(err)
	}

	failing := &reportRepositoryStub{err: errors.New("insert failed")}
	if _, err := NewReportService(failing, confirmedPlot(plotID), &surroundingsStub{}, metrics, testInstanceID).
		Create(context.Background(), testCaller, plotID); err == nil {
		t.Fatal("Create() succeeded with a failing repository")
	}

	if !slices.Equal(metrics.created, []string{"available unavailable"}) {
		t.Fatalf("created = %v", metrics.created)
	}
}

func TestExportServiceCountsBuiltFilesOnly(t *testing.T) {
	metrics := &metricsSpy{}
	withBoundary := domain.Report{ID: uuid.NewString(), Boundary: []domain.Coordinate{{}, {}, {}}}
	exports := NewExportService(reportReaderStub{report: withBoundary}, documentsStub{}, metrics)
	olderReport := NewExportService(
		reportReaderStub{report: domain.Report{ID: uuid.NewString()}},
		documentsStub{},
		metrics,
	)

	if _, err := exports.PDF(context.Background(), "owner", withBoundary.ID, "uk"); err != nil {
		t.Fatal(err)
	}

	if _, err := exports.GeoJSON(context.Background(), "owner", withBoundary.ID, "en"); err != nil {
		t.Fatal(err)
	}

	if _, err := olderReport.GeoJSON(
		context.Background(),
		"owner",
		"id",
		"uk",
	); !errors.Is(
		err,
		domain.ErrBoundaryNotRecorded,
	) {
		t.Fatalf("GeoJSON() error = %v", err)
	}

	if _, err := exports.PDF(context.Background(), "owner", withBoundary.ID, "de"); err == nil {
		t.Fatal("PDF() accepted an unknown language")
	}

	if !slices.Equal(metrics.downloads, []string{domain.ExtensionPDF, domain.ExtensionGeoJSON}) {
		t.Fatalf("downloads = %v", metrics.downloads)
	}
}

func TestOutboxRelayCountsPublishedAndFailedMessages(t *testing.T) {
	metrics := &metricsSpy{}
	published := &outboxRepositoryStub{pending: []domain.OutboxMessage{{ID: "event-1"}, {ID: "event-2"}}}
	newMeasuredRelay(
		published,
		messagePublisherStub{},
		metrics,
		slog.New(slog.DiscardHandler),
	).relayOnce(context.Background())

	unavailable := &outboxRepositoryStub{pending: []domain.OutboxMessage{{ID: "event-3"}}}
	newMeasuredRelay(unavailable, messagePublisherStub{err: errRelayBroker}, metrics, slog.New(slog.DiscardHandler)).
		relayOnce(context.Background())

	abandoned := &outboxRepositoryStub{
		pending:   []domain.OutboxMessage{{ID: "event-4"}},
		abandoned: &AbandonedMessage{ID: "event-4", Attempts: testMaxAttempts},
	}
	newMeasuredRelay(
		abandoned,
		messagePublisherStub{err: domain.ErrMessageRejected},
		metrics,
		slog.New(slog.DiscardHandler),
	).
		relayOnce(context.Background())

	if metrics.published != 2 || !slices.Equal(metrics.failures, []bool{false, true}) {
		t.Fatalf("published = %d, failures = %v", metrics.published, metrics.failures)
	}
}
