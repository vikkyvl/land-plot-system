package service

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

const testOwnerID = "owner-1"

var errBrokerUnavailable = errors.New("broker unavailable")

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func bufferLogger(logs *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(logs, nil))
}

type plotRepositoryStub struct {
	created         domain.Plot
	plot            domain.Plot
	plots           []domain.Plot
	updated         domain.Plot
	expectedVersion int64
	err             error
}

func (stub *plotRepositoryStub) Create(_ context.Context, plot domain.Plot) error {
	stub.created = plot
	return stub.err
}

func (stub *plotRepositoryStub) GetByID(context.Context, string) (domain.Plot, error) {
	return stub.plot, stub.err
}

func (stub *plotRepositoryStub) ListByOwner(context.Context, string) ([]domain.Plot, error) {
	return stub.plots, stub.err
}

func (stub *plotRepositoryStub) Update(_ context.Context, plot domain.Plot, expectedVersion int64) error {
	stub.updated = plot
	stub.expectedVersion = expectedVersion

	return stub.err
}

type eventPublisherStub struct {
	published []domain.PlotEvent
	err       error
}

func (stub *eventPublisherStub) Publish(_ context.Context, event domain.PlotEvent) error {
	stub.published = append(stub.published, event)
	return stub.err
}

func testPoints() []domain.Coordinate {
	return []domain.Coordinate{
		{Latitude: 50, Longitude: 30},
		{Latitude: 50, Longitude: 30.001},
		{Latitude: 50.001, Longitude: 30},
	}
}

func testPlotDraft() domain.PlotDraft {
	return domain.PlotDraft{OwnerID: testOwnerID, Name: "Plot", Points: testPoints()}
}

func TestPlotServiceCreateAssignsOwnerAndVersion(t *testing.T) {
	repository := &plotRepositoryStub{}
	publisher := &eventPublisherStub{}
	plots := NewPlotService(repository, publisher, domain.LocalEllipsoidCalculator{}, discardLogger())
	fixedNow := time.Date(2026, time.October, 26, 12, 0, 0, 0, time.UTC)
	plots.now = func() time.Time { return fixedNow }

	plot, err := plots.Create(context.Background(), testPlotDraft())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := uuid.Parse(plot.ID); err != nil {
		t.Fatalf("Create() ID = %q: %v", plot.ID, err)
	}

	if plot.OwnerID != testOwnerID || plot.Version != 1 || repository.created.ID != plot.ID {
		t.Fatalf("Create() plot = %#v", plot)
	}

	if len(publisher.published) != 1 {
		t.Fatalf("published events = %#v", publisher.published)
	}

	event := publisher.published[0]
	if event.Type != domain.PlotCreated || event.Plot.ID != plot.ID || !event.OccurredAt.Equal(fixedNow) {
		t.Fatalf("published event = %#v", event)
	}

	if _, err := uuid.Parse(event.ID); err != nil {
		t.Fatalf("event ID = %q: %v", event.ID, err)
	}
}

func TestPlotServiceRejectsForeignOwner(t *testing.T) {
	plotID := uuid.NewString()
	repository := &plotRepositoryStub{plot: domain.Plot{ID: plotID, OwnerID: testOwnerID}}
	plots := NewPlotService(repository, &eventPublisherStub{}, domain.LocalEllipsoidCalculator{}, discardLogger())

	_, err := plots.Get(context.Background(), "owner-2", plotID)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("Get() error = %v, want ErrForbidden", err)
	}
}

func TestPlotServiceRejectsInvalidIDBeforeRepository(t *testing.T) {
	repository := &plotRepositoryStub{err: errors.New("repository must not be called")}
	plots := NewPlotService(repository, &eventPublisherStub{}, domain.LocalEllipsoidCalculator{}, discardLogger())

	_, err := plots.Get(context.Background(), testOwnerID, "not-a-uuid")
	if !errors.Is(err, domain.ErrInvalidPlotID) {
		t.Fatalf("Get() error = %v, want ErrInvalidPlotID", err)
	}
}

func TestPlotServiceUpdateUsesOptimisticVersion(t *testing.T) {
	plotID := uuid.NewString()
	repository := &plotRepositoryStub{plot: domain.Plot{
		ID: plotID, OwnerID: testOwnerID, Name: "Plot", Points: testPoints(), Version: 3,
	}}
	publisher := &eventPublisherStub{}
	plots := NewPlotService(repository, publisher, domain.LocalEllipsoidCalculator{}, discardLogger())
	name := "Updated"

	updated, err := plots.Update(context.Background(), testOwnerID, plotID, domain.PlotChanges{Name: &name})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.Version != 4 || repository.expectedVersion != 3 || repository.updated.Name != name {
		t.Fatalf("Update() plot=%#v expectedVersion=%d", updated, repository.expectedVersion)
	}

	if len(publisher.published) != 1 || publisher.published[0].Type != domain.PlotUpdated ||
		publisher.published[0].Plot.Version != 4 {
		t.Fatalf("published events = %#v", publisher.published)
	}
}

func TestPlotServiceCreateSucceedsWhenPublishFailsAfterCommit(t *testing.T) {
	repository := &plotRepositoryStub{}

	var logs bytes.Buffer

	plots := NewPlotService(
		repository,
		&eventPublisherStub{err: errBrokerUnavailable},
		domain.LocalEllipsoidCalculator{},
		bufferLogger(&logs),
	)

	plot, err := plots.Create(context.Background(), testPlotDraft())
	if err != nil {
		t.Fatalf("Create() error = %v, want success after commit", err)
	}

	if repository.created.ID == "" || repository.created.ID != plot.ID {
		t.Fatalf("committed plot = %#v, returned plot = %#v", repository.created, plot)
	}

	assertPublishFailureLogged(t, logs.String(), plot.ID, domain.PlotCreated)
}

func TestPlotServiceUpdateSucceedsWhenPublishFailsAfterCommit(t *testing.T) {
	plotID := uuid.NewString()
	repository := &plotRepositoryStub{plot: domain.Plot{
		ID: plotID, OwnerID: testOwnerID, Name: "Plot", Points: testPoints(), Version: 3,
	}}

	var logs bytes.Buffer

	plots := NewPlotService(
		repository,
		&eventPublisherStub{err: errBrokerUnavailable},
		domain.LocalEllipsoidCalculator{},
		bufferLogger(&logs),
	)
	name := "Updated"

	updated, err := plots.Update(context.Background(), testOwnerID, plotID, domain.PlotChanges{Name: &name})
	if err != nil {
		t.Fatalf("Update() error = %v, want success after commit", err)
	}

	if updated.Version != 4 || repository.updated.Version != 4 {
		t.Fatalf("committed plot = %#v, returned plot = %#v", repository.updated, updated)
	}

	assertPublishFailureLogged(t, logs.String(), plotID, domain.PlotUpdated)
}

func assertPublishFailureLogged(t *testing.T, logs, plotID string, eventType domain.PlotEventType) {
	t.Helper()

	for _, want := range []string{
		"level=ERROR",
		`msg="publish plot event"`,
		"plot_id=" + plotID,
		"event_type=" + string(eventType),
		`error="broker unavailable"`,
	} {
		if !strings.Contains(logs, want) {
			t.Fatalf("logs = %q, want %q", logs, want)
		}
	}
}
