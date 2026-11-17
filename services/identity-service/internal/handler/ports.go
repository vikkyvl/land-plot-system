package handler

import (
	"context"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/domain"
)

type IdentityService interface {
	Profile(requestContext context.Context, principal domain.Principal) (domain.User, error)
}
