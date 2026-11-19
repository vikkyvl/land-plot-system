package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

func createdPlotWithSaga(t *testing.T) (*fakeRepository, domain.Plot, string) {
	t.Helper()

	repository := newFakeRepository()

	plot, err := newTestPlotService(repository).Create(context.Background(), testPlotDraft())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	return repository, plot, repository.committed.events[0].ID
}

func TestSagaServiceConfirmCompletesSagaAndConfirmsPlot(t *testing.T) {
	repository, plot, sagaID := createdPlotWithSaga(t)

	if err := NewSagaService(repository, discardLogger()).Confirm(context.Background(), sagaID); err != nil {
		t.Fatalf("Confirm() error = %v", err)
	}

	if repository.committed.plots[plot.ID].Status != domain.StatusConfirmed {
		t.Fatalf("plot = %#v", repository.committed.plots[plot.ID])
	}

	if repository.committed.sagas[sagaID].Status != domain.SagaCompleted {
		t.Fatalf("saga = %#v", repository.committed.sagas[sagaID])
	}
}

func TestSagaServiceIgnoresConfirmationOfUnknownSaga(t *testing.T) {
	repository, plot, _ := createdPlotWithSaga(t)

	if err := NewSagaService(repository, discardLogger()).Confirm(context.Background(), uuid.NewString()); err != nil {
		t.Fatalf("Confirm() error = %v", err)
	}

	if repository.committed.plots[plot.ID].Status != domain.StatusPending {
		t.Fatalf("plot = %#v", repository.committed.plots[plot.ID])
	}
}

func TestSagaServiceCompensatesCreateFromDeadLetter(t *testing.T) {
	repository, plot, sagaID := createdPlotWithSaga(t)

	err := NewSagaService(repository, discardLogger()).Compensate(context.Background(), sagaID, domain.ReasonDeadLetter)
	if err != nil {
		t.Fatalf("Compensate() error = %v", err)
	}

	cancelled := repository.committed.plots[plot.ID]
	if cancelled.Status != domain.StatusCancelled || cancelled.Version != 2 {
		t.Fatalf("plot = %#v", cancelled)
	}

	if repository.committed.sagas[sagaID].Status != domain.SagaCompensated ||
		repository.committed.reasons[sagaID] != domain.ReasonDeadLetter {
		t.Fatalf("saga = %#v, reason = %s", repository.committed.sagas[sagaID], repository.committed.reasons[sagaID])
	}

	events := repository.committed.events
	if len(events) != 2 || events[1].Type != domain.PlotCancelled || events[1].Plot.Version != 2 {
		t.Fatalf("outbox events = %#v", events)
	}
}

func TestSagaServiceRepeatedCompensationChangesNothing(t *testing.T) {
	repository, plot, sagaID := createdPlotWithSaga(t)
	sagas := NewSagaService(repository, discardLogger())

	for range 2 {
		if err := sagas.Compensate(context.Background(), sagaID, domain.ReasonDeadLetter); err != nil {
			t.Fatalf("Compensate() error = %v", err)
		}
	}

	if repository.committed.plots[plot.ID].Version != 2 || len(repository.committed.events) != 2 {
		t.Fatalf("plot = %#v, events = %d", repository.committed.plots[plot.ID], len(repository.committed.events))
	}
}

func TestSagaServiceIgnoresConfirmationAfterCompensation(t *testing.T) {
	repository, plot, sagaID := createdPlotWithSaga(t)
	sagas := NewSagaService(repository, discardLogger())

	if err := sagas.Compensate(context.Background(), sagaID, domain.ReasonTimeout); err != nil {
		t.Fatalf("Compensate() error = %v", err)
	}

	if err := sagas.Confirm(context.Background(), sagaID); err != nil {
		t.Fatalf("Confirm() error = %v", err)
	}

	if repository.committed.plots[plot.ID].Status != domain.StatusCancelled ||
		repository.committed.sagas[sagaID].Status != domain.SagaCompensated {
		t.Fatalf("plot = %#v, saga = %#v", repository.committed.plots[plot.ID], repository.committed.sagas[sagaID])
	}
}

func TestSagaServiceCompensatesUpdateByRestoringPreviousState(t *testing.T) {
	plotID := uuid.NewString()
	repository := newFakeRepository(storedPlot(plotID, domain.StatusConfirmed))
	name := "FAIL renamed"

	_, err := newTestPlotService(repository).Update(
		context.Background(),
		testOwnerID,
		plotID,
		domain.PlotChanges{Name: &name},
	)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	sagaID := repository.committed.events[0].ID

	err = NewSagaService(repository, discardLogger()).Compensate(context.Background(), sagaID, domain.ReasonDeadLetter)
	if err != nil {
		t.Fatalf("Compensate() error = %v", err)
	}

	restored := repository.committed.plots[plotID]
	if restored.Name != "Plot" || restored.Version != 5 || restored.Status != domain.StatusConfirmed {
		t.Fatalf("restored plot = %#v", restored)
	}

	events := repository.committed.events
	if len(events) != 2 || events[1].Type != domain.PlotUpdated || events[1].Plot.Name != "Plot" {
		t.Fatalf("outbox events = %#v", events)
	}
}

func TestSagaServiceCompensatesExpiredSagasWithTimeoutReason(t *testing.T) {
	repository, plot, sagaID := createdPlotWithSaga(t)
	repository.expiredSagaIDs = []string{sagaID}

	compensated, err := NewSagaService(repository, discardLogger()).CompensateExpired(context.Background())
	if err != nil || compensated != 1 {
		t.Fatalf("CompensateExpired() = %d, %v", compensated, err)
	}

	if repository.committed.plots[plot.ID].Status != domain.StatusCancelled ||
		repository.committed.reasons[sagaID] != domain.ReasonTimeout {
		t.Fatalf("plot = %#v, reason = %s", repository.committed.plots[plot.ID], repository.committed.reasons[sagaID])
	}
}

func createSagas(t *testing.T, repository *fakeRepository, count int) []string {
	t.Helper()

	plots := newTestPlotService(repository)
	for range count {
		if _, err := plots.Create(context.Background(), testPlotDraft()); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	sagaIDs := make([]string, 0, count)
	for _, event := range repository.committed.events {
		sagaIDs = append(sagaIDs, event.ID)
	}

	return sagaIDs
}

func TestSagaServiceCompensatesEachExpiredSagaSeparately(t *testing.T) {
	repository := newFakeRepository()
	sagaIDs := createSagas(t, repository, 3)
	repository.expiredSagaIDs = sagaIDs
	repository.failSagaID = sagaIDs[1]

	compensated, err := NewSagaService(repository, discardLogger()).CompensateExpired(context.Background())
	if !errors.Is(err, errSagaStore) || compensated != 2 {
		t.Fatalf("CompensateExpired() = %d, %v, want 2 and the store error", compensated, err)
	}

	for i, want := range []domain.SagaStatus{domain.SagaCompensated, domain.SagaStarted, domain.SagaCompensated} {
		if got := repository.committed.sagas[sagaIDs[i]].Status; got != want {
			t.Fatalf("saga %d status = %s, want %s", i, got, want)
		}
	}
}

func TestSagaServiceSkipsExpiredSagaFinishedElsewhere(t *testing.T) {
	repository := newFakeRepository()
	sagaIDs := createSagas(t, repository, 2)
	sagas := NewSagaService(repository, discardLogger())

	if err := sagas.Confirm(context.Background(), sagaIDs[0]); err != nil {
		t.Fatalf("Confirm() error = %v", err)
	}

	repository.expiredSagaIDs = sagaIDs

	compensated, err := sagas.CompensateExpired(context.Background())
	if err != nil || compensated != 1 {
		t.Fatalf("CompensateExpired() = %d, %v, want 1 without error", compensated, err)
	}

	if repository.committed.sagas[sagaIDs[0]].Status != domain.SagaCompleted {
		t.Fatalf("finished saga changed: %#v", repository.committed.sagas[sagaIDs[0]])
	}
}
