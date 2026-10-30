package handler

import (
	"time"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/handler/middleware"
	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/service"
)

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type loginResponse struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   time.Time `json:"expires_at"`
}

func newUserResponse(user domain.User) userResponse {
	return userResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt}
}

func newLoginResponse(token service.Token) loginResponse {
	return loginResponse{AccessToken: token.Value, TokenType: middleware.BearerScheme, ExpiresAt: token.ExpiresAt}
}
