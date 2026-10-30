package handler

import (
	"net/http"
	"testing"
	"time"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/handler/respond"
)

func TestCreatePlotReturnsCreatedPlot(t *testing.T) {
	plots := &plotServiceStub{plot: domain.Plot{
		ID: "plot-1", OwnerID: testOwnerID, Name: "Plot",
		Points:  []domain.Coordinate{{Latitude: 50, Longitude: 30}},
		Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}}

	response := serve(newTestRouter(plots), authorizedRequest(http.MethodPost, "/api/v1/plots", testPlotBody))

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}

	body := decodeBody[plotResponse](t, response)
	if body.ID != "plot-1" || len(body.Points) != 1 || body.Points[0].Latitude != 50 {
		t.Fatalf("response = %#v", body)
	}
}

func TestCreatePlotRejectsUnknownFields(t *testing.T) {
	request := authorizedRequest(http.MethodPost, "/api/v1/plots", `{"name":"Plot","points":[],"extra":1}`)

	response := serve(newTestRouter(&plotServiceStub{}), request)

	if response.Code != http.StatusBadRequest ||
		decodeBody[respond.ErrorResponse](t, response).Code != codeInvalidRequest {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
}

func TestUpdatePlotPassesOnlyProvidedFields(t *testing.T) {
	plots := &plotServiceStub{}
	request := authorizedRequest(http.MethodPatch, "/api/v1/plots/plot-1", `{"name":"Renamed"}`)

	response := serve(newTestRouter(plots), request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}

	if plots.changes.Name == nil || *plots.changes.Name != "Renamed" || plots.changes.Points != nil {
		t.Fatalf("changes = %#v", plots.changes)
	}
}

func TestListPlotsReturnsArray(t *testing.T) {
	response := serve(newTestRouter(&plotServiceStub{}), authorizedRequest(http.MethodGet, "/api/v1/plots", ""))

	if response.Code != http.StatusOK || len(decodeBody[[]plotResponse](t, response)) != 1 {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
}
