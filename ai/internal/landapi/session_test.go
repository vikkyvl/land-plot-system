package landapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestAuthenticateRequestsAClientCredentialsToken(t *testing.T) {
	gateway := &fakeGateway{mux: http.NewServeMux()}
	gateway.mux.HandleFunc("POST "+testTokenPath, func(response http.ResponseWriter, request *http.Request) {
		gateway.tokens.Add(1)

		if request.Header.Get("Content-Type") != contentTypeForm ||
			request.FormValue("grant_type") != "client_credentials" ||
			request.FormValue("client_id") != testClientID ||
			request.FormValue("client_secret") != testClientSecret ||
			request.FormValue("scope") != "plots:read plots:write" {
			writeJSON(response, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
			return
		}

		writeJSON(response, http.StatusOK, map[string]any{"access_token": "producer-token", "expires_in": 900})
	})

	client := gateway.start(t, "plots:read", "plots:write")

	if err := client.Authenticate(context.Background()); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	if client.token != "producer-token" || !client.expiresAt.Equal(testNow.Add(900*time.Second)) ||
		gateway.tokens.Load() != 1 {
		t.Fatalf("token = %q, expires at %v, requests = %d", client.token, client.expiresAt, gateway.tokens.Load())
	}
}

func TestAuthenticateReturnsTheOAuthError(t *testing.T) {
	gateway := &fakeGateway{mux: http.NewServeMux()}
	gateway.mux.HandleFunc("POST "+testTokenPath, func(response http.ResponseWriter, _ *http.Request) {
		writeJSON(response, http.StatusUnauthorized, map[string]string{
			"error":             "invalid_client",
			"error_description": "Invalid client or Invalid client credentials",
		})
	})

	err := gateway.start(t).Authenticate(context.Background())

	apiError, ok := errors.AsType[*APIError](err)
	if !ok || apiError.Status != http.StatusUnauthorized || apiError.Code != "invalid_client" ||
		apiError.Message != "Invalid client or Invalid client credentials" {
		t.Fatalf("Authenticate() error = %#v", err)
	}
}

func TestAuthenticateRejectsBrokenTokenResponses(t *testing.T) {
	for name, reply := range map[string]struct {
		status int
		body   string
	}{
		"no access token": {http.StatusOK, `{"expires_in":900}`},
		"not JSON":        {http.StatusOK, `<html>`},
		"plain error":     {http.StatusServiceUnavailable, `Service Unavailable`},
	} {
		t.Run(name, func(t *testing.T) {
			gateway := &fakeGateway{mux: http.NewServeMux()}
			gateway.mux.HandleFunc("POST "+testTokenPath, func(response http.ResponseWriter, _ *http.Request) {
				response.WriteHeader(reply.status)
				_, _ = response.Write([]byte(reply.body))
			})

			client := gateway.start(t)
			if err := client.Authenticate(context.Background()); err == nil || client.token != "" {
				t.Fatalf("Authenticate() error = %v, token = %q", err, client.token)
			}
		})
	}
}

func TestProducerScopesCannotCreateReports(t *testing.T) {
	gateway := newFakeGateway()
	gateway.mux.HandleFunc("GET /api/v1/plots", func(response http.ResponseWriter, _ *http.Request) {
		writeJSON(response, http.StatusOK, []Plot{})
	})
	gateway.mux.HandleFunc("POST /api/v1/reports", func(response http.ResponseWriter, request *http.Request) {
		if !strings.Contains(request.Header.Get("Authorization"), "reports:write") {
			response.Header().
				Set("WWW-Authenticate", `Bearer error="insufficient_scope", scope="reports:write plots:read"`)
			writeError(response, http.StatusForbidden, "insufficient_scope")

			return
		}

		writeJSON(response, http.StatusCreated, Report{ID: testReportID})
	})

	producer := gateway.start(t, "plots:read", "plots:write")

	if _, err := producer.ListPlots(context.Background()); err != nil {
		t.Fatalf("ListPlots() error = %v", err)
	}

	_, err := producer.CreateReport(context.Background(), testPlotID)

	apiError, ok := errors.AsType[*APIError](err)
	if !ok || apiError.Status != http.StatusForbidden || apiError.Code != "insufficient_scope" ||
		gateway.tokens.Load() != 1 {
		t.Fatalf("CreateReport() error = %#v, token requests = %d", err, gateway.tokens.Load())
	}
}
