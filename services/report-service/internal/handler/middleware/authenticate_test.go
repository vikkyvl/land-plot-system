package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/handler/respond"
)

type verifierStub struct {
	userID string
	err    error
}

func (stub verifierStub) Verify(string) (string, error) { return stub.userID, stub.err }

func authenticate(t *testing.T, verifier TokenVerifier, authorization string) (int, string) {
	t.Helper()

	var userID string

	next := http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		userID = PrincipalFrom(request.Context()).UserID
	})
	request := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	request.Header.Set("Authorization", authorization)

	response := httptest.NewRecorder()

	Authenticate(verifier, respond.NewWriter(slog.New(slog.DiscardHandler)), next).ServeHTTP(response, request)

	return response.Code, userID
}

func TestAuthenticateStoresVerifiedUserID(t *testing.T) {
	status, userID := authenticate(t, verifierStub{userID: "owner-1"}, "Bearer valid-token")

	if status != http.StatusOK || userID != "owner-1" {
		t.Fatalf("status = %d user ID = %q", status, userID)
	}
}

func TestAuthenticateRejectsMissingOrInvalidToken(t *testing.T) {
	tests := map[string]struct {
		verifier      TokenVerifier
		authorization string
	}{
		"missing header": {verifierStub{userID: "owner-1"}, ""},
		"wrong scheme":   {verifierStub{userID: "owner-1"}, "Basic credentials"},
		"invalid token":  {verifierStub{err: errors.New("bad token")}, "Bearer invalid-token"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			status, userID := authenticate(t, test.verifier, test.authorization)
			if status != http.StatusUnauthorized || userID != "" {
				t.Fatalf("status = %d user ID = %q", status, userID)
			}
		})
	}
}
