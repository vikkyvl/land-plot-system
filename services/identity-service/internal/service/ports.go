package service

import (
	"context"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/domain"
)

type UserRepository interface {
	Upsert(requestContext context.Context, user domain.User) (domain.User, error)
}

type IdentityMetrics interface {
	ProfileServed(newUser bool)
}
