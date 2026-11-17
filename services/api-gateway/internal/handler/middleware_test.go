package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/proxy"
)

type verifierStub struct {
	userID string
	email  string
	scopes []string
	err    error
}

func (stub verifierStub) Verify(string) (proxy.Principal, error) {
	return proxy.Principal{Subject: stub.userID, Email: stub.email, Scopes: stub.scopes}, stub.err
}

type limiterStub struct {
	allowed    bool
	retryAfter time.Duration
	keys       []string
}

func (stub *limiterStub) Allow(key string) (bool, time.Duration) {
	stub.keys = append(stub.keys, key)

	return stub.allowed, stub.retryAfter
}

func newTestRouter(upstream http.Handler, verifier TokenVerifier) http.Handler {
	return newTestRouterWithLimiter(upstream, verifier, &limiterStub{allowed: true})
}

func newTestRouterWithLimiter(upstream http.Handler, verifier TokenVerifier, limiter RequestLimiter) http.Handler {
	return NewRouter(slog.New(slog.DiscardHandler), "api-gateway", upstream, verifier, limiter, testOrigins)
}

func serve(router http.Handler, request *http.Request) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	return response
}

func TestProtectedRouteRequiresBearerToken(t *testing.T) {
	upstreamCalled := false
	upstream := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { upstreamCalled = true })

	response := serve(
		newTestRouter(upstream, verifierStub{}),
		httptest.NewRequest(http.MethodGet, "/api/v1/plots", http.NoBody),
	)

	if response.Code != http.StatusUnauthorized || upstreamCalled {
		t.Fatalf("status=%d upstreamCalled=%v", response.Code, upstreamCalled)
	}
}

func TestProtectedRouteRejectsInvalidToken(t *testing.T) {
	upstream := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("upstream called") })
	request := httptest.NewRequest(http.MethodGet, "/api/v1/users/profile", http.NoBody)
	request.Header.Set("Authorization", "Bearer invalid-token")

	response := serve(newTestRouter(upstream, verifierStub{err: errors.New("invalid")}), request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestProtectedRouteReplacesClientIdentityHeaders(t *testing.T) {
	upstream := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get(HeaderUserID) != "trusted-user" ||
			request.Header.Get(HeaderUserEmail) != "student@example.com" {
			t.Errorf("identity headers = %q, %q", request.Header.Get(HeaderUserID), request.Header.Get(HeaderUserEmail))
		}

		if value := request.Header.Get("X-User-Role"); value != "" {
			t.Errorf("X-User-Role = %q, want empty", value)
		}

		response.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/plots", http.NoBody)
	request.Header.Set("Authorization", "Bearer valid-token")
	request.Header.Set(HeaderUserID, "attacker")
	request.Header["x-user-role"] = []string{"admin"}

	response := serve(
		newTestRouter(upstream, verifierStub{
			userID: "trusted-user", email: "student@example.com", scopes: []string{proxy.ScopePlotsRead},
		}),
		request,
	)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestRequestIDIsKeptOrGenerated(t *testing.T) {
	router := newTestRouter(http.NotFoundHandler(), verifierStub{})
	request := httptest.NewRequest(http.MethodGet, "/health", http.NoBody)
	request.Header.Set(proxy.HeaderRequestID, "client-id")

	if got := serve(router, request).Header().Get(proxy.HeaderRequestID); got != "client-id" {
		t.Fatalf("kept request ID = %q", got)
	}

	generated := serve(
		router,
		httptest.NewRequest(http.MethodGet, "/health", http.NoBody),
	).Header().
		Get(proxy.HeaderRequestID)
	if len(generated) != 2*requestIDBytes {
		t.Fatalf("generated request ID = %q", generated)
	}
}

func authorized(method, target string) *http.Request {
	request := httptest.NewRequest(method, target, http.NoBody)
	request.Header.Set("Authorization", "Bearer valid-token")

	return request
}

func TestMissingTokenGetsInvalidTokenChallenge(t *testing.T) {
	response := serve(newTestRouter(http.NotFoundHandler(), verifierStub{}),
		httptest.NewRequest(http.MethodGet, "/api/v1/plots", http.NoBody))

	body := decodeError(t, response)
	if response.Code != http.StatusUnauthorized || body.Code != "unauthorized" ||
		response.Header().Get("WWW-Authenticate") != `Bearer error="invalid_token"` {
		t.Fatalf(
			"status = %d, body = %#v, challenge = %q",
			response.Code,
			body,
			response.Header().Get("WWW-Authenticate"),
		)
	}
}

func TestMissingScopeIsForbiddenBeforeProxying(t *testing.T) {
	upstream := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("upstream called") })
	router := newTestRouter(upstream, verifierStub{userID: "user-1", scopes: []string{proxy.ScopePlotsRead}})

	response := serve(router, authorized(http.MethodPost, "/api/v1/plots"))

	body := decodeError(t, response)
	if response.Code != http.StatusForbidden || body.Code != "insufficient_scope" || body.RequestID == "" ||
		response.Header().Get("WWW-Authenticate") != `Bearer error="insufficient_scope", scope="plots:write"` {
		t.Fatalf(
			"status = %d, body = %#v, challenge = %q",
			response.Code,
			body,
			response.Header().Get("WWW-Authenticate"),
		)
	}
}

func TestReportCreationNeedsBothScopes(t *testing.T) {
	router := newTestRouter(
		http.NotFoundHandler(),
		verifierStub{userID: "user-1", scopes: []string{proxy.ScopeReportsWrite}},
	)

	response := serve(router, authorized(http.MethodPost, "/api/v1/reports"))
	if response.Code != http.StatusForbidden ||
		response.Header().
			Get("WWW-Authenticate") !=
			`Bearer error="insufficient_scope", scope="reports:write plots:read"` {
		t.Fatalf("status = %d, challenge = %q", response.Code, response.Header().Get("WWW-Authenticate"))
	}
}

func TestUnknownMethodOnProtectedPathIsNotProxied(t *testing.T) {
	upstream := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("upstream called") })
	router := newTestRouter(upstream, verifierStub{userID: "user-1", scopes: []string{proxy.ScopePlotsWrite}})

	if response := serve(
		router,
		authorized(http.MethodDelete, "/api/v1/plots/plot-1"),
	); response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestDemoChainNeedsATokenWithoutScopes(t *testing.T) {
	upstream := http.HandlerFunc(
		func(response http.ResponseWriter, _ *http.Request) { response.WriteHeader(http.StatusNoContent) },
	)
	router := newTestRouter(upstream, verifierStub{userID: "user-1"})

	if response := serve(
		router,
		httptest.NewRequest(http.MethodGet, "/api/v1/demo/chain", http.NoBody),
	); response.Code != http.StatusUnauthorized {
		t.Fatalf("without token status = %d", response.Code)
	}

	if response := serve(
		router,
		authorized(http.MethodGet, "/api/v1/demo/chain"),
	); response.Code != http.StatusNoContent {
		t.Fatalf("with token status = %d", response.Code)
	}
}

func decodeError(t *testing.T, response *httptest.ResponseRecorder) struct {
	Code      string `json:"code"`
	RequestID string `json:"request_id"`
} {
	t.Helper()

	var body struct {
		Code      string `json:"code"`
		RequestID string `json:"request_id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}

	return body
}
