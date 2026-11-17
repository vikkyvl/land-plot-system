package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/handler/middleware"
)

const (
	testServiceName = "identity-service"
	testUserID      = "164d085d-f431-498a-819a-e5a1ec00e1c1"
)

type identityServiceStub struct {
	user      domain.User
	err       error
	principal domain.Principal
}

func (stub *identityServiceStub) Profile(_ context.Context, principal domain.Principal) (domain.User, error) {
	stub.principal = principal
	return stub.user, stub.err
}

type verifierStub struct {
	userID string
	scopes []string
	err    error
}

func (stub verifierStub) Verify(string) (domain.Principal, error) {
	return domain.Principal{Subject: stub.userID, Scopes: stub.scopes}, stub.err
}

func newTestRouterWithVerifier(identity IdentityService, verifier middleware.TokenVerifier) http.Handler {
	return NewRouter(slog.New(slog.DiscardHandler), testServiceName, identity, verifier)
}

func newTestRouter(identity IdentityService) http.Handler {
	return newTestRouterWithVerifier(identity, verifierStub{userID: testUserID, scopes: []string{domain.ScopeOpenID}})
}

func serve(router http.Handler, request *http.Request) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	return response
}

func decodeBody[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()

	var body T
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	return body
}
