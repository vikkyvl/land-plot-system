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
		Version: 1, Status: domain.StatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}}

	response := serve(newTestRouter(plots), authorizedRequest(http.MethodPost, "/api/v1/plots", testPlotBody))

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}

	body := decodeBody[plotResponse](t, response)
	if body.ID != "plot-1" || body.Status != "PENDING" || len(body.Points) != 1 || body.Points[0].Latitude != 50 {
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

func TestUpdatePendingPlotReturnsConflict(t *testing.T) {
	plots := &plotServiceStub{err: domain.ErrPlotNotConfirmed}
	request := authorizedRequest(http.MethodPatch, "/api/v1/plots/plot-1", `{"name":"Renamed"}`)

	response := serve(newTestRouter(plots), request)

	if response.Code != http.StatusConflict ||
		decodeBody[respond.ErrorResponse](t, response).Code != "plot_not_confirmed" {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
}

func TestDeletePlotReturnsNoContent(t *testing.T) {
	plots := &plotServiceStub{}

	response := serve(newTestRouter(plots), authorizedRequest(http.MethodDelete, "/api/v1/plots/plot-1", ""))

	if response.Code != http.StatusNoContent || response.Body.Len() != 0 ||
		plots.deletedID != "plot-1" || plots.ownerID != testOwnerID {
		t.Fatalf("status = %d body = %q deleted = %q owner = %q",
			response.Code, response.Body.String(), plots.deletedID, plots.ownerID)
	}
}

func TestDeletePlotErrors(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{domain.ErrPlotDeleted, http.StatusGone, "plot_deleted"},
		{domain.ErrPlotNotConfirmed, http.StatusConflict, "plot_not_confirmed"},
		{domain.ErrForbidden, http.StatusForbidden, "forbidden"},
	}
	for _, test := range tests {
		t.Run(test.wantCode, func(t *testing.T) {
			response := serve(newTestRouter(&plotServiceStub{err: test.err}),
				authorizedRequest(http.MethodDelete, "/api/v1/plots/plot-1", ""))

			if response.Code != test.wantStatus ||
				decodeBody[respond.ErrorResponse](t, response).Code != test.wantCode {
				t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestDeletePlotNeedsWriteScope(t *testing.T) {
	response := serve(newTestRouterWithScopes(&plotServiceStub{}, domain.ScopePlotsRead),
		authorizedRequest(http.MethodDelete, "/api/v1/plots/plot-1", ""))

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestPlotVersionsReturnsNewestFirstWithUnits(t *testing.T) {
	createdAt := time.Date(2026, time.October, 27, 10, 0, 0, 0, time.UTC)
	plots := &plotServiceStub{versions: []domain.PlotVersion{
		{
			Version: 2, Name: "Plot", AreaM2: 24_810, PerimeterM: 612, CreatedAt: createdAt,
			Points: []domain.Coordinate{{Latitude: 50.45, Longitude: 30.52}},
		},
		{Version: 1, Name: "Plot", AreaM2: 24_100, PerimeterM: 598, CreatedAt: createdAt.Add(-48 * time.Hour)},
	}}

	response := serve(newTestRouter(plots), authorizedRequest(http.MethodGet, "/api/v1/plots/plot-1/versions", ""))

	body := decodeBody[[]plotVersionResponse](t, response)
	if response.Code != http.StatusOK || len(body) != 2 || body[0].Version != 2 || body[0].AreaHectares != 2.481 ||
		body[0].AreaAres != 248.1 || len(body[0].Points) != 1 || body[1].Points == nil || plots.ownerID != testOwnerID {
		t.Fatalf("status = %d body = %#v", response.Code, body)
	}
}

func TestPlotVersionsOfUnknownPlotReturnEmptyArrayNotNull(t *testing.T) {
	response := serve(newTestRouter(&plotServiceStub{}),
		authorizedRequest(http.MethodGet, "/api/v1/plots/plot-1/versions", ""))

	if response.Code != http.StatusOK || response.Body.String() != "[]\n" {
		t.Fatalf("status = %d body = %q", response.Code, response.Body.String())
	}
}

func TestPlotVersionsNeedReadScope(t *testing.T) {
	response := serve(newTestRouterWithScopes(&plotServiceStub{}, domain.ScopePlotsWrite),
		authorizedRequest(http.MethodGet, "/api/v1/plots/plot-1/versions", ""))

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}
