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

func TestReportShowsWhetherItsPlotWasDeleted(t *testing.T) {
	deletedAt := time.Date(2026, time.November, 17, 12, 0, 0, 0, time.UTC)

	for _, test := range []struct {
		report domain.Report
		want   bool
	}{
		{domain.Report{ID: "report-1"}, false},
		{domain.Report{ID: "report-1", PlotDeletedAt: &deletedAt}, true},
	} {
		response := serve(newTestRouter(&reportServiceStub{report: test.report}),
			authorizedRequest(http.MethodGet, "/api/v1/reports/report-1", ""))

		if body := decodeBody[reportResponse](t, response); body.PlotDeleted != test.want {
			t.Fatalf("plot_deleted = %v, want %v", body.PlotDeleted, test.want)
		}
	}
}

func TestReportForDeletedPlotIsGone(t *testing.T) {
	response := serve(newTestRouter(&reportServiceStub{err: domain.ErrPlotDeleted}),
		authorizedRequest(http.MethodPost, "/api/v1/reports", `{"plot_id":"7d1f0a2e-3b4c-4d5e-8f60-718293a4b5c6"}`))

	if response.Code != http.StatusGone || decodeBody[respond.ErrorResponse](t, response).Code != "plot_deleted" {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
}

func TestListReportsPassesQueryAndOwner(t *testing.T) {
	reports := &reportServiceStub{reports: []domain.Report{{ID: "report-2"}, {ID: "report-1"}}}

	response := serve(newTestRouter(reports),
		authorizedRequest(http.MethodGet, "/api/v1/reports?plot_id=plot-9&limit=5", ""))

	body := decodeBody[[]reportResponse](t, response)
	if response.Code != http.StatusOK || len(body) != 2 || body[0].ID != "report-2" ||
		reports.ownerID != testOwnerID || reports.query != (service.ReportQuery{PlotID: "plot-9", Limit: "5"}) {
		t.Fatalf("status = %d body = %#v query = %#v", response.Code, body, reports.query)
	}
}

func TestListReportsWithoutReportsReturnsEmptyArray(t *testing.T) {
	response := serve(newTestRouter(&reportServiceStub{}), authorizedRequest(http.MethodGet, "/api/v1/reports", ""))

	if response.Code != http.StatusOK || response.Body.String() != "[]\n" {
		t.Fatalf("status = %d body = %q", response.Code, response.Body.String())
	}
}

func TestListReportsMapsInvalidQuery(t *testing.T) {
	response := serve(newTestRouter(&reportServiceStub{err: domain.ErrInvalidQuery}),
		authorizedRequest(http.MethodGet, "/api/v1/reports?limit=500", ""))

	if response.Code != http.StatusBadRequest ||
		decodeBody[respond.ErrorResponse](t, response).Code != "invalid_query" {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
}
