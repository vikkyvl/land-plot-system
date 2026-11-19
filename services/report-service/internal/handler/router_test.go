package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/handler/respond"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/service"
)

func TestReportRoutesRequireBearerToken(t *testing.T) {
	tests := map[string]string{
		http.MethodPost: "/api/v1/reports",
		http.MethodGet:  "/api/v1/reports/report-1",
	}
	for method, target := range tests {
		response := serve(newTestRouter(&reportServiceStub{}), httptest.NewRequest(method, target, http.NoBody))

		if response.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s status = %d, want %d", method, target, response.Code, http.StatusUnauthorized)
		}
	}
}

func TestServiceErrorReachesTheClient(t *testing.T) {
	request := authorizedRequest(http.MethodPost, "/api/v1/reports", `{"plot_id":"plot-1"}`)
	request.Header.Set(respond.RequestIDHeader, "request-1")

	response := serve(newTestRouter(&reportServiceStub{err: service.ErrPlotServiceUnavailable}), request)

	body := decodeBody[respond.ErrorResponse](t, response)
	if response.Code != http.StatusServiceUnavailable || body.Code != "plot_service_unavailable" ||
		body.RequestID != "request-1" {
		t.Fatalf("status = %d body = %#v", response.Code, body)
	}
}

func TestInvalidPlotResponseIsInternalError(t *testing.T) {
	reports := &reportServiceStub{err: service.ErrInvalidPlotResponse}

	response := serve(newTestRouter(reports), authorizedRequest(http.MethodPost, "/api/v1/reports", `{"plot_id":"p"}`))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}
