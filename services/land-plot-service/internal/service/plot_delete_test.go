package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

func TestPlotServiceDeleteMarksPlotAndAppendsEventInOneTransaction(t *testing.T) {
	plotID := uuid.NewString()
	repository := newFakeRepository(storedPlot(plotID, domain.StatusConfirmed))
	plots := newTestPlotService(repository)
	fixedNow := time.Date(2026, time.November, 17, 12, 0, 0, 0, time.UTC)
	plots.now = func() time.Time { return fixedNow }

	if err := plots.Delete(context.Background(), testOwnerID, plotID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	deleted := repository.committed.plots[plotID]
	if deleted.DeletedAt == nil || !deleted.DeletedAt.Equal(fixedNow) || deleted.Version != 4 ||
		deleted.Status != domain.StatusConfirmed {
		t.Fatalf("deleted plot = %#v", deleted)
	}

	events := repository.committed.events
	if len(events) != 1 || events[0].Type != domain.PlotDeleted || events[0].Plot.Version != 4 {
		t.Fatalf("outbox events = %#v", events)
	}

	if len(repository.committed.sagas) != 0 {
		t.Fatalf("delete started a saga: %#v", repository.committed.sagas)
	}
}

func TestPlotServiceDeleteRejectsPendingPlot(t *testing.T) {
	plotID := uuid.NewString()
	repository := newFakeRepository(storedPlot(plotID, domain.StatusPending))

	err := newTestPlotService(repository).Delete(context.Background(), testOwnerID, plotID)

	if !errors.Is(err, domain.ErrPlotNotConfirmed) || len(repository.committed.events) != 0 {
		t.Fatalf("Delete() error = %v, events = %#v", err, repository.committed.events)
	}
}

func TestPlotServiceDeleteRejectsForeignOwner(t *testing.T) {
	plotID := uuid.NewString()
	repository := newFakeRepository(storedPlot(plotID, domain.StatusConfirmed))

	err := newTestPlotService(repository).Delete(context.Background(), "owner-2", plotID)

	if !errors.Is(err, domain.ErrForbidden) || repository.committed.plots[plotID].DeletedAt != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestPlotServiceDeleteRollsBackWhenOutboxInsertFails(t *testing.T) {
	plotID := uuid.NewString()
	repository := newFakeRepository(storedPlot(plotID, domain.StatusConfirmed))
	repository.failAppendEvent = true

	err := newTestPlotService(repository).Delete(context.Background(), testOwnerID, plotID)

	if !errors.Is(err, errOutboxInsert) || repository.committed.plots[plotID].DeletedAt != nil {
		t.Fatalf("Delete() error = %v, plot = %#v", err, repository.committed.plots[plotID])
	}
}

func TestDeletedPlotIsGoneForReadAndUpdate(t *testing.T) {
	plotID := uuid.NewString()
	deletedAt := time.Date(2026, time.November, 17, 12, 0, 0, 0, time.UTC)
	plot := storedPlot(plotID, domain.StatusConfirmed)
	plot.DeletedAt = &deletedAt
	plots := newTestPlotService(newFakeRepository(plot))
	name := "Renamed"

	_, getErr := plots.Get(context.Background(), testOwnerID, plotID)
	_, updateErr := plots.Update(context.Background(), testOwnerID, plotID, domain.PlotChanges{Name: &name})
	deleteErr := plots.Delete(context.Background(), testOwnerID, plotID)

	for _, err := range []error{getErr, updateErr, deleteErr} {
		if !errors.Is(err, domain.ErrPlotDeleted) {
			t.Fatalf("errors = %v, %v, %v; want ErrPlotDeleted", getErr, updateErr, deleteErr)
		}
	}

	if _, err := plots.Get(context.Background(), "owner-2", plotID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("foreign Get() error = %v, want ErrForbidden", err)
	}
}
