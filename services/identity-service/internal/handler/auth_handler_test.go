package handler

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/handler/middleware"
	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/handler/respond"
	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/service"
)

func TestRegisterReturnsUserWithoutPasswordHash(t *testing.T) {
	identity := &identityServiceStub{
		user: domain.User{ID: testUserID, Email: "student@example.com", PasswordHash: "secret-hash"},
	}

	response := serve(newTestRouter(identity), postJSON("/api/v1/auth/register", testCredentialsBody))

	if response.Code != http.StatusCreated || strings.Contains(response.Body.String(), "secret-hash") {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}

	if body := decodeBody[userResponse](t, response); body.ID != testUserID {
		t.Fatalf("response = %#v", body)
	}
}

func TestRegisterRejectsUnknownJSONField(t *testing.T) {
	body := `{"email":"student@example.com","password":"password123","role":"admin"}`

	response := serve(newTestRouter(&identityServiceStub{}), postJSON("/api/v1/auth/register", body))

	if response.Code != http.StatusBadRequest ||
		decodeBody[respond.ErrorResponse](t, response).Code != errInvalidCredentialsBody.Code {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
}

func TestLoginReturnsBearerToken(t *testing.T) {
	expiresAt := time.Date(2026, time.October, 26, 13, 0, 0, 0, time.UTC)
	identity := &identityServiceStub{token: service.Token{Value: "access-token", ExpiresAt: expiresAt}}

	response := serve(newTestRouter(identity), postJSON("/api/v1/auth/login", testCredentialsBody))

	body := decodeBody[loginResponse](t, response)
	if response.Code != http.StatusOK || body.AccessToken != "access-token" ||
		body.TokenType != middleware.BearerScheme ||
		!body.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("status = %d body = %#v", response.Code, body)
	}
}
