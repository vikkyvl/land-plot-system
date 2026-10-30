package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/handler/respond"
)

func TestPlotRoutesRequireBearerToken(t *testing.T) {
	for _, target := range []string{"/api/v1/plots", "/api/v1/plots/plot-1", "/internal/plots/plot-1"} {
		response := serve(newTestRouter(&plotServiceStub{}), httptest.NewRequest(http.MethodGet, target, http.NoBody))

		if response.Code != http.StatusUnauthorized {
			t.Fatalf("GET %s status = %d, want %d", target, response.Code, http.StatusUnauthorized)
		}
	}
}

func TestOwnerComesFromVerifiedTokenNotHeader(t *testing.T) {
	plots := &plotServiceStub{}
	request := authorizedRequest(http.MethodPost, "/api/v1/plots", testPlotBody)
	request.Header.Set("X-User-ID", "forged-owner")

	serve(newTestRouter(plots), request)

	if plots.ownerID != testOwnerID {
		t.Fatalf("owner ID = %q, want %q", plots.ownerID, testOwnerID)
	}
}

func TestServiceErrorsReachTheClient(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{domain.ErrSelfIntersection, http.StatusUnprocessableEntity, "self_intersection"},
		{domain.ErrVersionConflict, http.StatusConflict, "version_conflict"},
	}
	for _, test := range tests {
		t.Run(test.wantCode, func(t *testing.T) {
			request := authorizedRequest(http.MethodPost, "/api/v1/plots", testPlotBody)
			request.Header.Set(respond.RequestIDHeader, "request-1")

			response := serve(newTestRouter(&plotServiceStub{err: test.err}), request)

			body := decodeBody[respond.ErrorResponse](t, response)
			if response.Code != test.wantStatus || body.Code != test.wantCode || body.RequestID != "request-1" {
				t.Fatalf("status = %d body = %#v", response.Code, body)
			}
		})
	}
}
