package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

func TestConfirmedCreateRecordsTheFirstVersion(t *testing.T) {
	repository, plot, sagaID := createdPlotWithSaga(t)

	if err := NewSagaService(
		repository,
		&metricsSpy{},
		discardLogger(),
	).Confirm(context.Background(), sagaID); err != nil {
		t.Fatalf("Confirm() error = %v", err)
	}

	versions := repository.committed.versions
	if len(versions) != 1 || versions[0].PlotID != plot.ID || versions[0].Version != 1 ||
		versions[0].Name != plot.Name || !versions[0].CreatedAt.Equal(plot.UpdatedAt) ||
		len(versions[0].Points) != len(plot.Points) {
		t.Fatalf("versions = %#v", versions)
	}
}

func TestCompensatedCreateRecordsNoVersion(t *testing.T) {
	repository, _, sagaID := createdPlotWithSaga(t)

	err := NewSagaService(
		repository,
		&metricsSpy{},
		discardLogger(),
	).Compensate(context.Background(), sagaID, domain.ReasonDeadLetter)
	if err != nil || len(repository.committed.versions) != 0 {
		t.Fatalf("Compensate() error = %v, versions = %#v", err, repository.committed.versions)
	}
}

func TestRestoredUpdateRecordsTheRestoredVersion(t *testing.T) {
	plotID := uuid.NewString()
	repository := newFakeRepository(storedPlot(plotID, domain.StatusConfirmed))
	name := "Renamed"

	if _, err := newTestPlotService(repository).Update(
		context.Background(), testOwnerID, plotID, domain.PlotChanges{Name: &name},
	); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	sagaID := repository.committed.events[0].ID

	err := NewSagaService(
		repository,
		&metricsSpy{},
		discardLogger(),
	).Compensate(context.Background(), sagaID, domain.ReasonTimeout)
	if err != nil {
		t.Fatalf("Compensate() error = %v", err)
	}

	versions := repository.committed.versions
	if len(versions) != 1 || versions[0].Version != 5 || versions[0].Name != "Plot" {
		t.Fatalf("versions = %#v", versions)
	}
}

func TestVersionsAreListedForTheOwnerOnly(t *testing.T) {
	plotID := uuid.NewString()
	repository := newFakeRepository(storedPlot(plotID, domain.StatusConfirmed))
	repository.committed.versions = []domain.PlotVersion{
		{PlotID: plotID, Version: 1}, {PlotID: uuid.NewString(), Version: 1}, {PlotID: plotID, Version: 2},
	}
	plots := newTestPlotService(repository)

	versions, err := plots.Versions(context.Background(), testOwnerID, plotID)
	if err != nil || len(versions) != 2 || versions[0].Version != 2 || versions[1].Version != 1 {
		t.Fatalf("Versions() = %#v, %v", versions, err)
	}

	if _, err := plots.Versions(context.Background(), "owner-2", plotID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("foreign Versions() error = %v, want ErrForbidden", err)
	}
}

func TestVersionsOfDeletedPlotAreGone(t *testing.T) {
	plotID := uuid.NewString()
	deletedAt := time.Date(2026, time.November, 18, 12, 0, 0, 0, time.UTC)
	plot := storedPlot(plotID, domain.StatusConfirmed)
	plot.DeletedAt = &deletedAt

	_, err := newTestPlotService(newFakeRepository(plot)).Versions(context.Background(), testOwnerID, plotID)
	if !errors.Is(err, domain.ErrPlotDeleted) {
		t.Fatalf("Versions() error = %v, want ErrPlotDeleted", err)
	}
}
