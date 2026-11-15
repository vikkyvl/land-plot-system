package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/handler/respond"
)

func profileRequest() *http.Request {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/users/profile", http.NoBody)
	request.Header.Set("Authorization", "Bearer valid-token")

	return request
}

func TestProfileRequiresBearerToken(t *testing.T) {
	response := serve(
		newTestRouter(&identityServiceStub{}),
		httptest.NewRequest(http.MethodGet, "/api/v1/users/profile", http.NoBody),
	)

	if response.Code != http.StatusUnauthorized ||
		decodeBody[respond.ErrorResponse](t, response).Code != "unauthorized" {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
}

func TestProfileRejectsInvalidToken(t *testing.T) {
	router := newTestRouterWithVerifier(&identityServiceStub{}, verifierStub{err: errors.New("bad token")})

	if response := serve(router, profileRequest()); response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestProfileUsesAuthenticatedSubject(t *testing.T) {
	identity := &identityServiceStub{user: domain.User{ID: testUserID, Email: "student@example.com"}}

	response := serve(newTestRouter(identity), profileRequest())

	if response.Code != http.StatusOK || identity.profileUserID != testUserID {
		t.Fatalf("status = %d profile user = %q", response.Code, identity.profileUserID)
	}

	if body := decodeBody[userResponse](t, response); body.Email != "student@example.com" {
		t.Fatalf("response = %#v", body)
	}
}
