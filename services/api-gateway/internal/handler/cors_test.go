package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/proxy"
)

const (
	allowedOrigin = "https://cad.kpi.ua"
	foreignOrigin = "https://evil.example"
)

var testOrigins = []string{allowedOrigin, "http://localhost:5173", "http://localhost:4173"}

func preflight(origin, method string) *http.Request {
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/plots", http.NoBody)
	request.Header.Set("Origin", origin)
	request.Header.Set("Access-Control-Request-Method", method)
	request.Header.Set("Access-Control-Request-Headers", "authorization, content-type")

	return request
}

func noUpstream(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("upstream called") })
}

func TestPreflightFromAnAllowedOriginNeedsNoToken(t *testing.T) {
	response := serve(newTestRouter(noUpstream(t), verifierStub{}), preflight(allowedOrigin, http.MethodPost))

	headers := response.Header()
	if response.Code != http.StatusNoContent ||
		headers.Get("Access-Control-Allow-Origin") != allowedOrigin ||
		headers.Get("Access-Control-Allow-Methods") != "GET, POST, PATCH, DELETE, OPTIONS" ||
		headers.Get(
			"Access-Control-Allow-Headers",
		) != "Authorization, Content-Type, X-Request-ID, traceparent, tracestate" ||
		headers.Get("Access-Control-Max-Age") != "600" ||
		headers.Get("Vary") != "Origin" ||
		headers.Get("Access-Control-Allow-Credentials") != "" ||
		headers.Get("Access-Control-Allow-Private-Network") != "" {
		t.Fatalf("status = %d, headers = %#v", response.Code, headers)
	}
}

func TestPreflightAllowsPrivateNetworkAccessWhenAsked(t *testing.T) {
	request := preflight(allowedOrigin, http.MethodGet)
	request.Header.Set("Access-Control-Request-Private-Network", "true")

	response := serve(newTestRouter(noUpstream(t), verifierStub{}), request)
	if response.Header().Get("Access-Control-Allow-Private-Network") != "true" {
		t.Fatalf("headers = %#v", response.Header())
	}
}

func TestPreflightFromAForeignOriginIsForbidden(t *testing.T) {
	for _, origin := range []string{foreignOrigin, "https://cad.kpi.ua.evil.com", "https://evil.cad.kpi.ua", "http://cad.kpi.ua", "null"} {
		t.Run(origin, func(t *testing.T) {
			response := serve(newTestRouter(noUpstream(t), verifierStub{}), preflight(origin, http.MethodGet))

			if response.Code != http.StatusForbidden || decodeError(t, response).Code != "origin_not_allowed" {
				t.Fatalf("status = %d", response.Code)
			}

			for name := range response.Header() {
				if strings.HasPrefix(name, "Access-Control-") {
					t.Fatalf("foreign origin got %s", name)
				}
			}
		})
	}
}

func TestRequestFromAnAllowedOriginGetsOneSetOfHeaders(t *testing.T) {
	upstream := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusOK)
	})
	request := authorized(http.MethodGet, "/api/v1/plots")
	request.Header.Set("Origin", allowedOrigin)

	response := serve(
		newTestRouter(upstream, verifierStub{userID: "user-1", scopes: []string{proxy.ScopePlotsRead}}),
		request,
	)

	if response.Code != http.StatusOK || len(response.Header().Values("Access-Control-Allow-Origin")) != 1 ||
		response.Header().Get("Access-Control-Allow-Origin") != allowedOrigin ||
		response.Header().
			Get("Access-Control-Expose-Headers") !=
			"X-Request-ID, X-Trace-ID, Retry-After, Content-Disposition" {
		t.Fatalf("status = %d, headers = %#v", response.Code, response.Header())
	}
}

func TestErrorsForAnAllowedOriginAreReadableByTheBrowser(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/plots", http.NoBody)
	request.Header.Set("Origin", allowedOrigin)

	response := serve(newTestRouter(noUpstream(t), verifierStub{}), request)
	if response.Code != http.StatusUnauthorized ||
		response.Header().Get("Access-Control-Allow-Origin") != allowedOrigin {
		t.Fatalf("status = %d, headers = %#v", response.Code, response.Header())
	}
}

func TestRequestFromAForeignOriginGetsNoCORSHeaders(t *testing.T) {
	upstream := http.HandlerFunc(
		func(response http.ResponseWriter, _ *http.Request) { response.WriteHeader(http.StatusOK) },
	)
	request := authorized(http.MethodGet, "/api/v1/plots")
	request.Header.Set("Origin", "https://cad.kpi.ua.evil.com")

	response := serve(
		newTestRouter(upstream, verifierStub{userID: "user-1", scopes: []string{proxy.ScopePlotsRead}}),
		request,
	)
	if response.Header().Get("Access-Control-Allow-Origin") != "" || response.Header().Get("Vary") != "Origin" {
		t.Fatalf("headers = %#v", response.Header())
	}
}

func TestRequestWithoutOriginIsUnchanged(t *testing.T) {
	upstream := http.HandlerFunc(
		func(response http.ResponseWriter, _ *http.Request) { response.WriteHeader(http.StatusOK) },
	)

	response := serve(newTestRouter(upstream, verifierStub{userID: "user-1", scopes: []string{proxy.ScopePlotsRead}}),
		authorized(http.MethodGet, "/api/v1/plots"))
	if response.Header().Get("Vary") != "" || response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("headers = %#v", response.Header())
	}
}
