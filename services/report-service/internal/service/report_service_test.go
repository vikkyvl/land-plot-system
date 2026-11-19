package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

const testInstanceID = "report-1"

var testCaller = Caller{UserID: "owner-1", AccessToken: "token", RequestID: "request-1"}

type reportRepositoryStub struct {
	created domain.Report
	report  domain.Report
	filter  domain.ReportFilter
	listed  bool
	err     error
}

func (stub *reportRepositoryStub) Create(_ context.Context, report domain.Report) error {
	stub.created = report
	return stub.err
}

func (stub *reportRepositoryStub) GetByID(context.Context, string) (domain.Report, error) {
	return stub.report, stub.err
}

func (stub *reportRepositoryStub) List(_ context.Context, filter domain.ReportFilter) ([]domain.Report, error) {
	stub.filter, stub.listed = filter, true

	return []domain.Report{stub.report}, stub.err
}

type surroundingsStub struct {
	plot domain.PlotSnapshot
}

func (stub *surroundingsStub) Describe(_ context.Context, plot domain.PlotSnapshot) (domain.Terrain, domain.Climate) {
	stub.plot = plot

	return domain.Terrain{
			Status:       domain.DataAvailable,
			SlopeDegrees: 4,
		}, domain.UnavailableClimate(
			domain.ClimatePeriod{},
		)
}

type plotClientStub struct {
	plot   domain.PlotSnapshot
	caller Caller
	plotID string
	err    error
}

func (stub *plotClientStub) Get(_ context.Context, caller Caller, plotID string) (domain.PlotSnapshot, error) {
	stub.caller = caller
	stub.plotID = plotID

	return stub.plot, stub.err
}

func (stub *plotClientStub) Trace(_ context.Context, caller Caller, _ string) ([]domain.TraceHop, error) {
	stub.caller = caller

	return []domain.TraceHop{{Service: "land-plot-service", Instance: "plot-1", Protocol: "HTTP"}}, stub.err
}

func TestReportServiceCreatesSnapshotFromPlotService(t *testing.T) {
	plotID := uuid.NewString()
	repository := &reportRepositoryStub{}
	plots := &plotClientStub{plot: domain.PlotSnapshot{
		ID: plotID, OwnerID: testCaller.UserID, Name: "Plot", AreaM2: 10_000, PerimeterM: 400, Version: 2,
		Status: domain.PlotStatusConfirmed,
	}}
	reports := NewReportService(repository, plots, &surroundingsStub{}, testInstanceID)
	fixedNow := time.Date(2026, time.October, 26, 12, 0, 0, 0, time.UTC)
	reports.now = func() time.Time { return fixedNow }

	report, err := reports.Create(context.Background(), testCaller, plotID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if report.AreaAres != 100 || report.AreaHectares != 1 || report.PlotVersion != 2 ||
		!report.CreatedAt.Equal(fixedNow) {
		t.Fatalf("Create() report = %#v", report)
	}

	if plots.caller != testCaller || plots.plotID != plotID || repository.created.ID != report.ID {
		t.Fatalf("downstream call or persistence mismatch: caller=%#v created=%#v", plots.caller, repository.created)
	}
}

func TestReportServiceRejectsInvalidPlotIDBeforeCallingPlotService(t *testing.T) {
	plots := &plotClientStub{}

	_, err := NewReportService(
		&reportRepositoryStub{},
		plots,
		&surroundingsStub{}, testInstanceID,
	).Create(context.Background(), testCaller, "bad")

	if !errors.Is(err, domain.ErrInvalidPlotID) || plots.plotID != "" {
		t.Fatalf("Create() error = %v, plot service called with %q", err, plots.plotID)
	}
}

func TestReportServiceRejectsForeignPlot(t *testing.T) {
	plotID := uuid.NewString()
	plots := &plotClientStub{
		plot: domain.PlotSnapshot{
			ID:         plotID,
			OwnerID:    "owner-2",
			Name:       "Plot",
			AreaM2:     100,
			PerimeterM: 40,
			Version:    1,
		},
	}

	_, err := NewReportService(
		&reportRepositoryStub{},
		plots,
		&surroundingsStub{}, testInstanceID,
	).Create(context.Background(), testCaller, plotID)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("Create() error = %v, want ErrForbidden", err)
	}
}

func TestReportServiceProtectsStoredReport(t *testing.T) {
	reportID := uuid.NewString()
	repository := &reportRepositoryStub{report: domain.Report{ID: reportID, OwnerID: testCaller.UserID}}

	_, err := NewReportService(
		repository,
		&plotClientStub{},
		&surroundingsStub{}, testInstanceID,
	).Get(context.Background(), "owner-2", reportID)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("Get() error = %v, want ErrForbidden", err)
	}
}

func TestReportServiceBuildsDiagnosticPath(t *testing.T) {
	plots := &plotClientStub{}
	reports := NewReportService(&reportRepositoryStub{}, plots, &surroundingsStub{}, testInstanceID)

	trace, err := reports.Trace(context.Background(), testCaller, "gateway-1", "hello")
	if err != nil {
		t.Fatalf("Trace() error = %v", err)
	}

	if len(trace.Path) != 3 || trace.Path[0].Instance != "gateway-1" || trace.Path[1].Instance != testInstanceID ||
		trace.Path[2].Instance != "plot-1" {
		t.Fatalf("Trace() = %#v", trace)
	}

	if plots.caller != testCaller {
		t.Fatalf("land-plot-service got caller %#v, want %#v", plots.caller, testCaller)
	}
}

func TestReportServiceRejectsPlotWaitingForConfirmation(t *testing.T) {
	plotID := uuid.NewString()
	repository := &reportRepositoryStub{}
	plots := &plotClientStub{plot: domain.PlotSnapshot{
		ID: plotID, OwnerID: testCaller.UserID, Name: "Plot", AreaM2: 100, PerimeterM: 40, Version: 1,
		Status: "PENDING",
	}}

	_, err := NewReportService(
		repository,
		plots,
		&surroundingsStub{},
		testInstanceID,
	).Create(context.Background(), testCaller, plotID)
	if !errors.Is(err, domain.ErrPlotNotConfirmed) || repository.created.ID != "" {
		t.Fatalf("Create() error = %v, created = %#v", err, repository.created)
	}
}

func TestReportServiceStoresAccuracyTerrainAndClimate(t *testing.T) {
	plotID := uuid.NewString()
	repository := &reportRepositoryStub{}
	accuracy := &domain.AreaAccuracy{AreaMarginM2: 520, Basis: "assumed"}
	plot := domain.PlotSnapshot{
		ID: plotID, OwnerID: testCaller.UserID, Name: "Plot", AreaM2: 10_000, PerimeterM: 400, Version: 2,
		Status: domain.PlotStatusConfirmed, Points: testTerrainBoundary, Accuracy: accuracy,
	}
	surroundings := &surroundingsStub{}

	report, err := NewReportService(repository, &plotClientStub{plot: plot}, surroundings, testInstanceID).
		Create(context.Background(), testCaller, plotID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if len(surroundings.plot.Points) != len(testTerrainBoundary) || report.Accuracy != accuracy ||
		report.Terrain == nil || report.Terrain.SlopeDegrees != 4 ||
		report.Climate == nil || report.Climate.Status != domain.DataUnavailable {
		t.Fatalf("report = %+v", report)
	}

	if repository.created.Terrain != report.Terrain || repository.created.Climate != report.Climate {
		t.Fatalf("stored report = %+v", repository.created)
	}
}
