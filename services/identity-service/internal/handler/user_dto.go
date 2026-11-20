package handler

import (
	"time"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/domain"
)

type profileResponse struct {
	Subject   string    `json:"sub"`
	Email     string    `json:"email,omitempty"`
	Name      string    `json:"name,omitempty"`
	Scopes    []string  `json:"scopes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func newProfileResponse(user domain.User, scopes []string) profileResponse {
	if scopes == nil {
		scopes = []string{}
	}

	return profileResponse{
		Subject:   user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Scopes:    scopes,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
