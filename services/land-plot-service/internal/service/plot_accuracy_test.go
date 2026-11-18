package service

import (
	"context"
	"errors"
	"testing"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

func TestPreviewEstimatesAccuracyOfValidPolygon(t *testing.T) {
	plots := newTestPlotService(newFakeRepository())
	capture := domain.Capture{Zoom: 18, Layer: domain.LayerMap, Pointer: domain.PointerMouse}

	inspection, err := plots.Preview(testPoints(), domain.Survey{Capture: &capture})
	if err != nil || !inspection.Valid() || inspection.Accuracy == nil {
		t.Fatalf("Preview() = %+v, %v", inspection, err)
	}

	worst, err := plots.Preview(testPoints(), domain.Survey{})
	if err != nil || worst.Accuracy.AreaMarginM2 <= inspection.Accuracy.AreaMarginM2 {
		t.Fatalf("fallback margin %v is not above the zoom 18 mouse margin %v", worst.Accuracy, inspection.Accuracy)
	}
}

func TestPreviewRejectsInvalidSurveyAndSkipsAccuracyOfInvalidPolygon(t *testing.T) {
	plots := newTestPlotService(newFakeRepository())

	_, err := plots.Preview(testPoints(), domain.Survey{Obstructions: []domain.Obstruction{{Side: 9, Reason: "trees"}}})
	if !errors.Is(err, domain.ErrInvalidObstruction) {
		t.Fatalf("Preview() error = %v, want ErrInvalidObstruction", err)
	}

	inspection, err := plots.Preview(testPoints()[:2], domain.Survey{})
	if err != nil || inspection.Valid() || inspection.Accuracy != nil {
		t.Fatalf("Preview() = %+v, %v", inspection, err)
	}
}

func TestCreatedAndReadPlotsCarryAccuracy(t *testing.T) {
	repository := newFakeRepository()
	plots := newTestPlotService(repository)

	created, err := plots.Create(context.Background(), testPlotDraft())
	if err != nil || created.Accuracy.AreaMarginM2 <= 0 || created.Accuracy.Confidence != domain.AccuracyConfidence {
		t.Fatalf("Create() = %+v, %v", created.Accuracy, err)
	}

	listed, err := plots.List(context.Background(), testOwnerID)
	if err != nil || len(listed) != 1 || listed[0].Accuracy != created.Accuracy {
		t.Fatalf("List() = %+v, %v", listed, err)
	}
}
