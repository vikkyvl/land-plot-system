package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

func TestTraceNeedsATokenButNoScopeAndForwardsIt(t *testing.T) {
	reports := &reportServiceStub{trace: domain.Trace{
		Message: "hello",
		Path:    []domain.TraceHop{{Service: "api-gateway", Instance: "gateway-1", Protocol: "HTTP"}},
	}}

	anonymous := serve(newTestRouter(reports), httptest.NewRequest(http.MethodGet, "/api/v1/demo/chain", http.NoBody))
	if anonymous.Code != http.StatusUnauthorized {
		t.Fatalf("without token status = %d", anonymous.Code)
	}

	router := newTestRouterWithVerifier(reports, verifierStub{userID: testOwnerID})
	response := serve(router, authorizedRequest(http.MethodGet, "/api/v1/demo/chain?message=hello", ""))

	body := decodeBody[traceResponse](t, response)
	if response.Code != http.StatusOK || len(body.Path) != 1 || body.Path[0].Instance != "gateway-1" {
		t.Fatalf("status = %d body = %#v", response.Code, body)
	}

	if reports.caller.AccessToken != testAccessToken || reports.caller.UserID != testOwnerID {
		t.Fatalf("caller = %#v", reports.caller)
	}
}

func TestTraceRejectsLongMessage(t *testing.T) {
	target := "/api/v1/demo/chain?message=" + strings.Repeat("x", maximumTraceMessageLength+1)

	response := serve(newTestRouter(&reportServiceStub{}), authorizedRequest(http.MethodGet, target, ""))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}
