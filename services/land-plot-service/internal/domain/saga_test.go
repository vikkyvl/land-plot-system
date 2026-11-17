package domain

import (
	"reflect"
	"testing"
	"time"
)

func confirmedTestPlot(t *testing.T) Plot {
	t.Helper()

	plot, err := NewPlot(
		"plot-1",
		PlotDraft{OwnerID: "owner-1", Name: "Original", Points: referencePolygon()},
		testCalculator,
		testNow,
	)
	if err != nil {
		t.Fatalf("NewPlot() error = %v", err)
	}

	return plot.Confirmed()
}

func TestCreateSagaCompensationCancelsPlot(t *testing.T) {
	plot, err := NewPlot(
		"plot-1",
		PlotDraft{OwnerID: "owner-1", Name: "Plot", Points: referencePolygon()},
		testCalculator,
		testNow,
	)
	if err != nil {
		t.Fatalf("NewPlot() error = %v", err)
	}

	saga := NewCreateSaga("event-1", plot)

	compensated, eventType := saga.Compensate(plot, testNow.Add(time.Minute))
	if compensated.Status != StatusCancelled || compensated.Version != 2 || eventType != PlotCancelled {
		t.Fatalf("Compensate() plot = %#v, event type = %s", compensated, eventType)
	}

	if saga.ID != "event-1" || saga.Kind != SagaCreate || saga.PreviousState != nil || saga.NewVersion != 1 {
		t.Fatalf("NewCreateSaga() = %#v", saga)
	}
}

func TestUpdateSagaCompensationRestoresPreviousStateWithNewVersion(t *testing.T) {
	original := confirmedTestPlot(t)
	name := "FAIL renamed"
	triangle := referencePolygon()[:3]

	updated, err := original.Updated(PlotChanges{Name: &name, Points: &triangle}, testCalculator, testNow)
	if err != nil {
		t.Fatalf("Updated() error = %v", err)
	}

	saga := NewUpdateSaga("event-2", original, updated)

	restored, eventType := saga.Compensate(updated, testNow.Add(time.Minute))
	if eventType != PlotUpdated || restored.Status != StatusConfirmed || restored.Version != 3 {
		t.Fatalf("Compensate() plot = %#v, event type = %s", restored, eventType)
	}

	if restored.Name != original.Name || restored.AreaM2 != original.AreaM2 ||
		!reflect.DeepEqual(restored.Points, original.Points) {
		t.Fatalf("restored = %#v, want state of %#v", restored, original)
	}

	if saga.NewVersion != 2 || saga.PreviousState.Version != 1 {
		t.Fatalf("NewUpdateSaga() = %#v", saga)
	}
}
