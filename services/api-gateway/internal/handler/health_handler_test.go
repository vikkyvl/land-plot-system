package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	response := serve(
		newTestRouter(http.NotFoundHandler(), nil),
		httptest.NewRequest(http.MethodGet, "/health", http.NoBody),
	)

	var body healthResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Code != http.StatusOK || body.Status != healthStatusOK || body.Service != "api-gateway" {
		t.Fatalf("status = %d body = %+v", response.Code, body)
	}
}
