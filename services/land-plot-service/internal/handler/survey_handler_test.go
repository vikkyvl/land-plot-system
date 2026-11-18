package handler

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

const (
	surveyFields      = `"capture":{"zoom":18,"layer":"satellite","pointer":"touch"},"obstructions":[{"side":1,"reason":"trees"}]`
	squarePointsField = `"points":[{"lat":50.45,"lon":30.52},{"lat":50.45,"lon":30.5214},` +
		`{"lat":50.4509,"lon":30.5214},{"lat":50.4509,"lon":30.52}]`
)

var expectedSurvey = domain.Survey{
	Capture:      &domain.Capture{Zoom: 18, Layer: domain.LayerSatellite, Pointer: domain.PointerTouch},
	Obstructions: []domain.Obstruction{{Side: 1, Reason: domain.ReasonTrees}},
}

func TestPreviewPassesSurveyAndReturnsAccuracy(t *testing.T) {
	plots := &plotServiceStub{}

	response := serve(newTestRouter(plots), previewRequest("{"+squarePointsField+","+surveyFields+"}"))

	body := decodeBody[previewPlotResponse](t, response)
	if response.Code != http.StatusOK || body.Accuracy == nil ||
		body.Accuracy.AreaM2Margin != stubAccuracy.AreaMarginM2 {
		t.Fatalf("status = %d body = %#v", response.Code, body)
	}

	if !reflect.DeepEqual(plots.survey, expectedSurvey) {
		t.Fatalf("survey = %+v, want %+v", plots.survey, expectedSurvey)
	}
}

func TestPreviewRejectsInvalidSurvey(t *testing.T) {
	tests := map[string]string{
		`"capture":{"zoom":18,"layer":"terrain","pointer":"touch"}`: "invalid_capture",
		`"obstructions":[{"side":4,"reason":"trees"}]`:              "invalid_obstruction",
		`"obstructions":[{"side":0,"reason":"fog"}]`:                "invalid_obstruction",
	}

	for field, code := range tests {
		response := serve(newTestRouter(&plotServiceStub{}), previewRequest("{"+squarePointsField+","+field+"}"))

		body := response.Body.String()
		if response.Code != http.StatusBadRequest || !strings.Contains(body, `"code":"`+code+`"`) {
			t.Errorf("%s: status = %d body = %s", field, response.Code, body)
		}

		if code == "invalid_obstruction" && !strings.Contains(body, `"details":{"index":0}`) {
			t.Errorf("%s: body = %s, want the index of the obstruction", field, body)
		}
	}
}

func TestCreatePlotPassesSurveyAndShowsAccuracy(t *testing.T) {
	worstSide := 1
	plots := &plotServiceStub{plot: domain.Plot{
		ID: "plot-1", Points: []domain.Coordinate{{Latitude: 50, Longitude: 30}},
		Accuracy: domain.Accuracy{AreaMarginM2: 520, WorstSide: &worstSide, Basis: domain.BasisAssumed},
	}}

	request := authorizedRequest(
		http.MethodPost,
		"/api/v1/plots",
		`{"name":"Plot",`+squarePointsField+","+surveyFields+"}",
	)
	response := serve(newTestRouter(plots), request)

	if response.Code != http.StatusCreated || !reflect.DeepEqual(plots.draft.Survey, expectedSurvey) {
		t.Fatalf("status = %d survey = %+v", response.Code, plots.draft.Survey)
	}

	var body map[string]json.RawMessage
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	accuracy := string(body["accuracy"])
	if string(body["capture"]) != "null" || string(body["obstructions"]) != "[]" ||
		!strings.Contains(accuracy, `"area_m2_margin":520`) || !strings.Contains(accuracy, `"worst_side":1`) ||
		!strings.Contains(accuracy, `"basis":"assumed"`) {
		t.Fatalf("body = %s", body)
	}
}

func TestUpdatePlotPassesSurveyChanges(t *testing.T) {
	plots := &plotServiceStub{}
	request := authorizedRequest(http.MethodPatch, "/api/v1/plots/plot-1", "{"+surveyFields+"}")

	response := serve(newTestRouter(plots), request)

	if response.Code != http.StatusOK || plots.changes.Capture == nil || plots.changes.Obstructions == nil ||
		!reflect.DeepEqual(*plots.changes.Obstructions, expectedSurvey.Obstructions) || plots.changes.Points != nil {
		t.Fatalf("status = %d changes = %+v", response.Code, plots.changes)
	}
}
