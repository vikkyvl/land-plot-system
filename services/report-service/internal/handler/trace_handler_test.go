package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

func TestTraceIsPublic(t *testing.T) {
	reports := &reportServiceStub{trace: domain.Trace{
		Message: "hello",
		Path:    []domain.TraceHop{{Service: "api-gateway", Instance: "gateway-1", Protocol: "HTTP"}},
	}}
	router := newTestRouterWithVerifier(reports, verifierStub{err: errors.New("must not verify")})

	response := serve(router, httptest.NewRequest(http.MethodGet, "/api/v1/demo/chain?message=hello", http.NoBody))

	body := decodeBody[traceResponse](t, response)
	if response.Code != http.StatusOK || len(body.Path) != 1 || body.Path[0].Instance != "gateway-1" {
		t.Fatalf("status = %d body = %#v", response.Code, body)
	}
}

func TestTraceRejectsLongMessage(t *testing.T) {
	target := "/api/v1/demo/chain?message=" + strings.Repeat("x", maximumTraceMessageLength+1)

	response := serve(newTestRouter(&reportServiceStub{}), httptest.NewRequest(http.MethodGet, target, http.NoBody))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}
