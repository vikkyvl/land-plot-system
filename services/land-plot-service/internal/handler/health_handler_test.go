package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	response := serve(newTestRouter(nil), httptest.NewRequest(http.MethodGet, "/health", http.NoBody))

	body := decodeBody[healthResponse](t, response)
	if response.Code != http.StatusOK || body.Status != healthStatusOK || body.Service != testServiceName {
		t.Fatalf("status = %d body = %+v", response.Code, body)
	}
}
