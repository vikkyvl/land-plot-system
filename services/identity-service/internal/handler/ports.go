package handler

import (
	"context"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/service"
)

type IdentityService interface {
	Register(requestContext context.Context, email, password string) (domain.User, error)
	Login(requestContext context.Context, email, password string) (service.Token, error)
	Profile(requestContext context.Context, userID string) (domain.User, error)
}
