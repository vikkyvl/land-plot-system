package service

import (
	"context"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

func TestPlotServiceCountsOnlyCommittedChanges(t *testing.T) {
	metrics := &metricsSpy{}
	plotID := uuid.NewString()
	repository := newFakeRepository(storedPlot(plotID, domain.StatusConfirmed))
	plots := newMeasuredPlotService(repository, metrics)
	name := "Renamed"

	if _, err := plots.Create(context.Background(), testPlotDraft()); err != nil {
		t.Fatal(err)
	}

	if _, err := plots.Update(context.Background(), testOwnerID, plotID, domain.PlotChanges{Name: &name}); err != nil {
		t.Fatal(err)
	}

	repository.failAppendEvent = true

	if _, err := plots.Create(context.Background(), testPlotDraft()); err == nil {
		t.Fatal("Create() succeeded with a failing outbox")
	}

	want := []domain.PlotEventType{domain.PlotCreated, domain.PlotUpdated}
	if !slices.Equal(metrics.plotChanges, want) {
		t.Fatalf("plot changes = %v, want %v", metrics.plotChanges, want)
	}
}

func TestPlotServiceCountsPreviewProblemsByCode(t *testing.T) {
	metrics := &metricsSpy{}
	plots := newMeasuredPlotService(newFakeRepository(), metrics)
	square := []domain.Coordinate{
		{Latitude: 50.45, Longitude: 30.52},
		{Latitude: 50.451, Longitude: 30.52},
		{Latitude: 50.451, Longitude: 30.522},
	}
	hourglass := []domain.Coordinate{
		{
			Latitude:  50.4509,
			Longitude: 30.52,
		},
		{Latitude: 50.4509, Longitude: 30.5214},
		{Latitude: 50.45, Longitude: 30.52},
		{Latitude: 50.45, Longitude: 30.5214},
	}
	badZoom := domain.Survey{Capture: &domain.Capture{Zoom: 30, Layer: domain.LayerMap, Pointer: domain.PointerMouse}}

	_, _ = plots.Preview(square, domain.Survey{})
	_, _ = plots.Preview(hourglass, domain.Survey{})
	_, _ = plots.Preview(square, badZoom)

	if len(metrics.previews) != 3 || len(metrics.previews[0]) != 0 ||
		!slices.Contains(metrics.previews[1], "self_intersection") ||
		!slices.Equal(metrics.previews[2], []string{"invalid_capture"}) {
		t.Fatalf("previews = %v", metrics.previews)
	}
}

func TestSagaServiceCountsConfirmationsOnce(t *testing.T) {
	repository, _, sagaID := createdPlotWithSaga(t)
	metrics := &metricsSpy{}
	sagas := NewSagaService(repository, metrics, discardLogger())

	for range 2 {
		if err := sagas.Confirm(context.Background(), sagaID); err != nil {
			t.Fatal(err)
		}
	}

	if !slices.Equal(metrics.confirmed, []domain.SagaKind{domain.SagaCreate}) {
		t.Fatalf("confirmed = %v", metrics.confirmed)
	}
}

func TestSagaServiceCountsCancelledAndRevertedSagas(t *testing.T) {
	repository, _, createSagaID := createdPlotWithSaga(t)
	metrics := &metricsSpy{}
	sagas := NewSagaService(repository, metrics, discardLogger())

	plotID := uuid.NewString()
	updates := newFakeRepository(storedPlot(plotID, domain.StatusConfirmed))
	name := "FAIL renamed"

	if _, err := newTestPlotService(updates).Update(
		context.Background(), testOwnerID, plotID, domain.PlotChanges{Name: &name},
	); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		if err := sagas.Compensate(context.Background(), createSagaID, domain.ReasonDeadLetter); err != nil {
			t.Fatal(err)
		}
	}

	err := NewSagaService(updates, metrics, discardLogger()).
		Compensate(context.Background(), updates.committed.events[0].ID, domain.ReasonTimeout)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"CREATE plot.cancelled dlq", "UPDATE plot.updated timeout"}
	if !slices.Equal(metrics.compensations, want) {
		t.Fatalf("compensations = %v, want %v", metrics.compensations, want)
	}
}

func TestOutboxRelayCountsPublishedAndFailedMessages(t *testing.T) {
	metrics := &metricsSpy{}
	published := &outboxRepositoryStub{pending: []domain.OutboxMessage{{ID: "event-1"}, {ID: "event-2"}}}
	newMeasuredRelay(published, messagePublisherStub{}, metrics, discardLogger()).relayOnce(context.Background())

	unavailable := &outboxRepositoryStub{pending: []domain.OutboxMessage{{ID: "event-3"}}}
	newMeasuredRelay(unavailable, messagePublisherStub{err: errBrokerUnavailable}, metrics, discardLogger()).
		relayOnce(context.Background())

	abandoned := &outboxRepositoryStub{
		pending:   []domain.OutboxMessage{{ID: "event-4"}},
		abandoned: &AbandonedMessage{ID: "event-4", Attempts: testMaxAttempts},
	}
	newMeasuredRelay(abandoned, messagePublisherStub{err: domain.ErrMessageRejected}, metrics, discardLogger()).
		relayOnce(context.Background())

	if metrics.published != 2 || !slices.Equal(metrics.failures, []bool{false, true}) {
		t.Fatalf("published = %d, failures = %v", metrics.published, metrics.failures)
	}
}
