package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

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

func TestProfileReturnsTheUserAndTheTokenScopes(t *testing.T) {
	now := time.Date(2026, time.November, 17, 12, 0, 0, 0, time.UTC)
	identity := &identityServiceStub{user: domain.User{
		ID: testUserID, Email: "taras.melnyk@example.com", Name: "Тарас Мельник", CreatedAt: now, UpdatedAt: now,
	}}

	response := serve(newTestRouter(identity), profileRequest())

	body := decodeBody[profileResponse](t, response)
	if response.Code != http.StatusOK || identity.principal.Subject != testUserID ||
		body.Subject != testUserID || body.Email != "taras.melnyk@example.com" || body.Name != "Тарас Мельник" ||
		!slices.Equal(body.Scopes, []string{domain.ScopeOpenID}) {
		t.Fatalf("status = %d, principal = %#v, body = %#v", response.Code, identity.principal, body)
	}
}

func TestProfileReportsAnInvalidSubject(t *testing.T) {
	response := serve(newTestRouter(&identityServiceStub{err: domain.ErrInvalidSubject}), profileRequest())

	if response.Code != http.StatusUnauthorized ||
		decodeBody[respond.ErrorResponse](t, response).Code != "invalid_subject" {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
}

func TestRegisterAndLoginAreGone(t *testing.T) {
	for _, target := range []string{"/api/v1/auth/register", "/api/v1/auth/login"} {
		request := httptest.NewRequest(
			http.MethodPost,
			target,
			strings.NewReader(`{"email":"a@b.c","password":"password123"}`),
		)
		if response := serve(newTestRouter(&identityServiceStub{}), request); response.Code != http.StatusNotFound {
			t.Fatalf("POST %s status = %d, want %d", target, response.Code, http.StatusNotFound)
		}
	}
}

func TestProfileRequiresTheOpenIDScope(t *testing.T) {
	router := newTestRouterWithVerifier(
		&identityServiceStub{},
		verifierStub{userID: testUserID, scopes: []string{"plots:read"}},
	)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/users/profile", http.NoBody)
	request.Header.Set("Authorization", "Bearer valid-token")

	response := serve(router, request)
	if response.Code != http.StatusForbidden ||
		response.Header().Get("WWW-Authenticate") != `Bearer error="insufficient_scope", scope="openid"` {
		t.Fatalf("status = %d, challenge = %q", response.Code, response.Header().Get("WWW-Authenticate"))
	}
}
