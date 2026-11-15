package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/handler/middleware"
	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/service"
)

const (
	testServiceName     = "identity-service"
	testUserID          = "user-1"
	testCredentialsBody = `{"email":"student@example.com","password":"password123"}`
)

type identityServiceStub struct {
	user          domain.User
	token         service.Token
	err           error
	profileUserID string
}

func (stub *identityServiceStub) Register(context.Context, string, string) (domain.User, error) {
	return stub.user, stub.err
}

func (stub *identityServiceStub) Login(context.Context, string, string) (service.Token, error) {
	return stub.token, stub.err
}

func (stub *identityServiceStub) Profile(_ context.Context, userID string) (domain.User, error) {
	stub.profileUserID = userID
	return stub.user, stub.err
}

type verifierStub struct {
	userID string
	err    error
}

func (stub verifierStub) Verify(string) (string, error) { return stub.userID, stub.err }

func newTestRouterWithVerifier(identity IdentityService, verifier middleware.TokenVerifier) http.Handler {
	return NewRouter(slog.New(slog.DiscardHandler), testServiceName, identity, verifier)
}

func newTestRouter(identity IdentityService) http.Handler {
	return newTestRouterWithVerifier(identity, verifierStub{userID: testUserID})
}

func postJSON(target, body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
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
