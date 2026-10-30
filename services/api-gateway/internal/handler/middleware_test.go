package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/proxy"
)

type verifierStub struct {
	userID string
	email  string
	err    error
}

func (stub verifierStub) Verify(string) (string, string, error) {
	return stub.userID, stub.email, stub.err
}

func newTestRouter(upstream http.Handler, verifier TokenVerifier) http.Handler {
	return NewRouter(slog.New(slog.DiscardHandler), "api-gateway", upstream, verifier)
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
		newTestRouter(upstream, verifierStub{userID: "trusted-user", email: "student@example.com"}),
		request,
	)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestPublicRouteSkipsAuthenticationButStripsIdentityHeaders(t *testing.T) {
	upstream := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if value := request.Header.Get(HeaderUserID); value != "" {
			t.Errorf("%s = %q, want empty", HeaderUserID, value)
		}

		response.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", http.NoBody)
	request.Header.Set(HeaderUserID, "attacker")

	response := serve(newTestRouter(upstream, verifierStub{err: errors.New("must not be called")}), request)

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
