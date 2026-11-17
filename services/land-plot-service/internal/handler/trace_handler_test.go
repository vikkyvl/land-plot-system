package handler

import (
	"net/http"
	"testing"
)

func TestTraceIncludesInstance(t *testing.T) {
	response := serve(
		newTestRouter(nil),
		authorizedRequest(http.MethodGet, "/internal/demo/chain?message=hi", ""),
	)

	body := decodeBody[traceResponse](t, response)
	if response.Code != http.StatusOK || body.Message != "hi" || len(body.Path) != 1 ||
		body.Path[0].Instance != testInstanceID {
		t.Fatalf("status = %d body = %#v", response.Code, body)
	}
}

func TestTraceUsesDefaultMessage(t *testing.T) {
	response := serve(newTestRouter(nil), authorizedRequest(http.MethodGet, "/internal/demo/chain", ""))

	if body := decodeBody[traceResponse](t, response); body.Message != defaultTraceMessage {
		t.Fatalf("message = %q, want %q", body.Message, defaultTraceMessage)
	}
}
