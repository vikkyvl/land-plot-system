package domain

import (
	"errors"
	"reflect"
	"testing"
)

var (
	testCapture = Capture{Zoom: 18, Layer: LayerSatellite, Pointer: PointerMouse}
	testSurvey  = Survey{Capture: &testCapture, Obstructions: []Obstruction{{Side: 1, Reason: ReasonTrees}}}
)

func surveyedPlot(t *testing.T) Plot {
	t.Helper()

	plot, err := NewPlot(
		"plot-1",
		PlotDraft{OwnerID: "owner-1", Name: "Plot", Points: referencePolygon(), Survey: testSurvey},
		testCalculator,
		testNow,
	)
	if err != nil {
		t.Fatalf("NewPlot() error = %v", err)
	}

	return plot.Confirmed()
}

func TestNewPlotKeepsSurveyAndRejectsInvalidOne(t *testing.T) {
	plot := surveyedPlot(t)
	if !reflect.DeepEqual(plot.Survey, testSurvey) || plot.Survey.Capture == testSurvey.Capture {
		t.Fatalf("survey = %+v, want a copy of %+v", plot.Survey, testSurvey)
	}

	outOfRange := Survey{Obstructions: []Obstruction{{Side: len(referencePolygon()), Reason: ReasonTrees}}}

	_, err := NewPlot(
		"plot-1",
		PlotDraft{OwnerID: "owner-1", Name: "Plot", Points: referencePolygon(), Survey: outOfRange},
		testCalculator,
		testNow,
	)
	if !errors.Is(err, ErrInvalidObstruction) {
		t.Fatalf("NewPlot() error = %v, want ErrInvalidObstruction", err)
	}
}

func TestUpdatedSurveyFollowsTheChanges(t *testing.T) {
	plot := surveyedPlot(t)
	name := "Renamed"
	points := referencePolygon()
	marks := []Obstruction{{Side: 0, Reason: ReasonUnclear}}

	tests := map[string]struct {
		changes PlotChanges
		want    Survey
	}{
		"name keeps the survey":           {PlotChanges{Name: &name}, testSurvey},
		"points without survey fall back": {PlotChanges{Points: &points}, Survey{}},
		"points take the new survey": {
			PlotChanges{Points: &points, Capture: &testCapture, Obstructions: &marks},
			Survey{Capture: &testCapture, Obstructions: marks},
		},
		"obstructions alone keep the capture": {
			PlotChanges{Obstructions: &marks},
			Survey{Capture: &testCapture, Obstructions: marks},
		},
	}

	for title, test := range tests {
		updated, err := plot.Updated(test.changes, testCalculator, testNow)
		if err != nil {
			t.Fatalf("%s: error = %v", title, err)
		}

		if !reflect.DeepEqual(updated.Survey, test.want) {
			t.Errorf("%s: survey = %+v, want %+v", title, updated.Survey, test.want)
		}
	}
}

func TestUpdatedRejectsCaptureWithoutPointsAndForeignSides(t *testing.T) {
	plot := surveyedPlot(t)

	if _, err := plot.Updated(PlotChanges{Capture: &testCapture}, testCalculator, testNow); !errors.Is(
		err,
		ErrInvalidCapture,
	) {
		t.Fatalf("capture alone: error = %v, want ErrInvalidCapture", err)
	}

	triangle := referencePolygon()[:3]
	marks := []Obstruction{{Side: 3, Reason: ReasonTrees}}

	_, err := plot.Updated(PlotChanges{Points: &triangle, Obstructions: &marks}, testCalculator, testNow)
	if !errors.Is(err, ErrInvalidObstruction) {
		t.Fatalf("side of a triangle: error = %v, want ErrInvalidObstruction", err)
	}
}

func TestCompensationRestoresTheSurvey(t *testing.T) {
	plot := surveyedPlot(t)
	points := referencePolygon()

	updated, err := plot.Updated(PlotChanges{Points: &points}, testCalculator, testNow)
	if err != nil {
		t.Fatalf("Updated() error = %v", err)
	}

	restored, _ := NewUpdateSaga("saga-1", plot, updated).Compensate(updated, testNow)
	if !reflect.DeepEqual(restored.Survey, testSurvey) {
		t.Fatalf("restored survey = %+v, want %+v", restored.Survey, testSurvey)
	}

	if !reflect.DeepEqual(plot.ConfirmedVersion().Survey, testSurvey) {
		t.Fatalf("version survey = %+v", plot.ConfirmedVersion().Survey)
	}
}
