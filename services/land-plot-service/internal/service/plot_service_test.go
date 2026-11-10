package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

const testSagaTimeout = 30 * time.Second

func newTestPlotService(repository *fakeRepository) *PlotService {
	return NewPlotService(repository, domain.LocalEllipsoidCalculator{}, testSagaTimeout)
}

func TestPlotServiceCreateWritesPlotSagaAndEventInOneTransaction(t *testing.T) {
	repository := newFakeRepository()
	plots := newTestPlotService(repository)
	fixedNow := time.Date(2026, time.October, 26, 12, 0, 0, 0, time.UTC)
	plots.now = func() time.Time { return fixedNow }

	plot, err := plots.Create(context.Background(), testPlotDraft())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := uuid.Parse(plot.ID); err != nil {
		t.Fatalf("Create() ID = %q: %v", plot.ID, err)
	}

	if plot.OwnerID != testOwnerID || plot.Version != 1 || plot.Status != domain.StatusPending {
		t.Fatalf("Create() plot = %#v", plot)
	}

	committed := repository.committed
	if len(committed.events) != 1 || committed.plots[plot.ID].ID != plot.ID {
		t.Fatalf("committed plots = %#v, events = %#v", committed.plots, committed.events)
	}

	event := committed.events[0]
	if event.Type != domain.PlotCreated || event.Plot.ID != plot.ID || !event.OccurredAt.Equal(fixedNow) {
		t.Fatalf("outbox event = %#v", event)
	}

	saga := committed.sagas[event.ID]
	if saga.Kind != domain.SagaCreate || saga.Status != domain.SagaStarted || saga.PlotID != plot.ID {
		t.Fatalf("saga = %#v", saga)
	}

	if repository.sagaTimeout != testSagaTimeout {
		t.Fatalf("saga timeout = %v, want %v", repository.sagaTimeout, testSagaTimeout)
	}
}

func TestPlotServiceCreateRollsBackPlotWhenOutboxInsertFails(t *testing.T) {
	repository := newFakeRepository()
	repository.failAppendEvent = true

	_, err := newTestPlotService(repository).Create(context.Background(), testPlotDraft())
	if !errors.Is(err, errOutboxInsert) {
		t.Fatalf("Create() error = %v, want outbox insert failure", err)
	}

	if len(repository.committed.plots) != 0 || len(repository.committed.sagas) != 0 {
		t.Fatalf("committed plots = %#v, sagas = %#v", repository.committed.plots, repository.committed.sagas)
	}
}

func TestPlotServiceRejectsForeignOwner(t *testing.T) {
	plotID := uuid.NewString()
	plots := newTestPlotService(newFakeRepository(storedPlot(plotID, domain.StatusConfirmed)))

	_, err := plots.Get(context.Background(), "owner-2", plotID)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("Get() error = %v, want ErrForbidden", err)
	}
}

func TestPlotServiceHidesCancelledPlot(t *testing.T) {
	plotID := uuid.NewString()
	plots := newTestPlotService(newFakeRepository(storedPlot(plotID, domain.StatusCancelled)))

	_, err := plots.Get(context.Background(), testOwnerID, plotID)
	if !errors.Is(err, domain.ErrPlotNotFound) {
		t.Fatalf("Get() error = %v, want ErrPlotNotFound", err)
	}
}

func TestPlotServiceRejectsInvalidIDBeforeRepository(t *testing.T) {
	repository := newFakeRepository()
	repository.err = errors.New("repository must not be called")

	_, err := newTestPlotService(repository).Get(context.Background(), testOwnerID, "not-a-uuid")
	if !errors.Is(err, domain.ErrInvalidPlotID) {
		t.Fatalf("Get() error = %v, want ErrInvalidPlotID", err)
	}
}

func TestPlotServiceUpdateStartsSagaWithPreviousState(t *testing.T) {
	plotID := uuid.NewString()
	repository := newFakeRepository(storedPlot(plotID, domain.StatusConfirmed))
	name := "Updated"

	updated, err := newTestPlotService(repository).Update(
		context.Background(),
		testOwnerID,
		plotID,
		domain.PlotChanges{Name: &name},
	)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.Version != 4 || updated.Status != domain.StatusPending ||
		repository.committed.plots[plotID].Name != name {
		t.Fatalf("Update() plot = %#v", updated)
	}

	events := repository.committed.events
	if len(events) != 1 || events[0].Type != domain.PlotUpdated || events[0].Plot.Version != 4 {
		t.Fatalf("outbox events = %#v", events)
	}

	saga := repository.committed.sagas[events[0].ID]
	if saga.Kind != domain.SagaUpdate || saga.PreviousState == nil ||
		saga.PreviousState.Name != "Plot" || saga.PreviousState.Version != 3 {
		t.Fatalf("saga = %#v", saga)
	}
}

func TestPlotServiceUpdateRejectsPendingPlot(t *testing.T) {
	plotID := uuid.NewString()
	repository := newFakeRepository(storedPlot(plotID, domain.StatusPending))
	name := "Updated"

	_, err := newTestPlotService(repository).Update(
		context.Background(),
		testOwnerID,
		plotID,
		domain.PlotChanges{Name: &name},
	)
	if !errors.Is(err, domain.ErrPlotNotConfirmed) {
		t.Fatalf("Update() error = %v, want ErrPlotNotConfirmed", err)
	}

	if len(repository.committed.events) != 0 || len(repository.committed.sagas) != 0 {
		t.Fatalf("committed events = %#v, sagas = %#v", repository.committed.events, repository.committed.sagas)
	}
}
