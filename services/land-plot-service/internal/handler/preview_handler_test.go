package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/handler/respond"
)

const (
	previewTarget    = "/api/v1/plots/preview"
	squareBody       = `{"points":[{"lat":50.45,"lon":30.52},{"lat":50.45,"lon":30.5214},{"lat":50.4509,"lon":30.5214},{"lat":50.4509,"lon":30.52}]}`
	hourglassBody    = `{"points":[{"lat":50.4509,"lon":30.52},{"lat":50.4509,"lon":30.5214},{"lat":50.45,"lon":30.52},{"lat":50.45,"lon":30.5214}]}`
	selfIntersection = `{"code":"self_intersection","message":"polygon must not intersect itself",` +
		`"details":{"segments":[[1,2],[3,0]],"swap":[2,3]}}`
)

func previewRequest(body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, previewTarget, strings.NewReader(body))
}

func TestPreviewWorksWithoutToken(t *testing.T) {
	response := serve(newTestRouter(&plotServiceStub{}), previewRequest(squareBody))

	body := decodeBody[previewPlotResponse](t, response)
	if response.Code != http.StatusOK || !body.Valid || len(body.Errors) != 0 ||
		body.AreaM2 == nil || body.PerimeterM == nil || body.AreaAres == nil || body.AreaHectares == nil {
		t.Fatalf("status = %d body = %#v", response.Code, body)
	}

	if *body.AreaAres != *body.AreaM2/domain.SquareMetersPerAre ||
		*body.AreaHectares != *body.AreaM2/domain.SquareMetersPerHectare {
		t.Fatalf("area = %v m², %v ares, %v ha", *body.AreaM2, *body.AreaAres, *body.AreaHectares)
	}
}

func TestPreviewOfInvalidPolygonIsOKWithErrors(t *testing.T) {
	response := serve(newTestRouter(&plotServiceStub{}), previewRequest(hourglassBody))

	var body map[string]json.RawMessage
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Code != http.StatusOK || string(body["valid"]) != "false" || string(body["area_m2"]) != "null" ||
		string(body["perimeter_m"]) != "null" || string(body["errors"]) != "["+selfIntersection+"]" {
		t.Fatalf("status = %d body = %s", response.Code, body)
	}
}

func TestPreviewOfEmptyPointsListsTheProblem(t *testing.T) {
	response := serve(newTestRouter(&plotServiceStub{}), previewRequest(`{"points":[]}`))

	body := decodeBody[previewPlotResponse](t, response)
	if response.Code != http.StatusOK || body.Valid || len(body.Errors) != 1 ||
		body.Errors[0].Code != "insufficient_points" {
		t.Fatalf("status = %d body = %#v", response.Code, body)
	}
}

func TestPreviewRejectsMalformedBodies(t *testing.T) {
	for _, body := range []string{`{}`, `{"points":`, `{"points":[],"name":"Plot"}`, `[]`} {
		response := serve(newTestRouter(&plotServiceStub{}), previewRequest(body))

		if response.Code != http.StatusBadRequest ||
			decodeBody[respond.ErrorResponse](t, response).Code != codeInvalidRequest {
			t.Fatalf("body %s: status = %d", body, response.Code)
		}
	}
}

func TestCreatePlotReturnsValidationDetails(t *testing.T) {
	plots := &plotServiceStub{
		err: domain.ErrDuplicatePoint.WithDetails(domain.DuplicatePointDetails{Indices: [2]int{0, 2}}),
	}

	response := serve(newTestRouter(plots), authorizedRequest(http.MethodPost, "/api/v1/plots", testPlotBody))

	if response.Code != http.StatusUnprocessableEntity ||
		!strings.Contains(response.Body.String(), `"code":"duplicate_point"`) ||
		!strings.Contains(response.Body.String(), `"details":{"indices":[0,2]}`) {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
}

func TestPlotResponseCarriesAresAndHectares(t *testing.T) {
	plots := &plotServiceStub{plot: domain.Plot{ID: "plot-1", AreaM2: 24_810}}

	response := serve(newTestRouter(plots), authorizedRequest(http.MethodGet, "/api/v1/plots/plot-1", ""))

	body := decodeBody[plotResponse](t, response)
	if body.AreaAres != 248.1 || body.AreaHectares != 2.481 {
		t.Fatalf("body = %#v", body)
	}
}
