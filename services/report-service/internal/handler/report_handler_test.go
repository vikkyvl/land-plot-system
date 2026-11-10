package handler

import (
	"net/http"
	"testing"
	"time"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/handler/respond"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/service"
)

func TestCreateReportPassesAuthenticatedCaller(t *testing.T) {
	reports := &reportServiceStub{report: domain.Report{
		ID: "report-1", PlotID: "plot-1", PlotName: "Plot", PlotVersion: 1,
		AreaM2: 10_000, AreaAres: 100, AreaHectares: 1, PerimeterM: 400, CreatedAt: time.Now(),
	}}
	request := authorizedRequest(http.MethodPost, "/api/v1/reports", `{"plot_id":"plot-1"}`)
	request.Header.Set(respond.RequestIDHeader, "request-1")

	response := serve(newTestRouter(reports), request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}

	want := service.Caller{UserID: testOwnerID, AccessToken: testAccessToken, RequestID: "request-1"}
	if reports.caller != want || reports.plotID != "plot-1" {
		t.Fatalf("caller = %#v plotID = %q", reports.caller, reports.plotID)
	}

	if body := decodeBody[reportResponse](t, response); body.ID != "report-1" || body.AreaHectares != 1 {
		t.Fatalf("response = %#v", body)
	}
}

func TestCreateReportRejectsInvalidBody(t *testing.T) {
	response := serve(
		newTestRouter(&reportServiceStub{}),
		authorizedRequest(http.MethodPost, "/api/v1/reports", `{"plot":"x"}`),
	)

	if response.Code != http.StatusBadRequest ||
		decodeBody[respond.ErrorResponse](t, response).Code != "invalid_request" {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
}

func TestGetReportUsesAuthenticatedOwner(t *testing.T) {
	reports := &reportServiceStub{}

	response := serve(newTestRouter(reports), authorizedRequest(http.MethodGet, "/api/v1/reports/report-1", ""))

	if response.Code != http.StatusOK || reports.ownerID != testOwnerID {
		t.Fatalf("status = %d owner = %q", response.Code, reports.ownerID)
	}
}
