package service

import (
	"context"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/domain"
)

type UserRepository interface {
	Create(requestContext context.Context, user domain.User) error
	FindByEmail(requestContext context.Context, email string) (domain.User, error)
	FindByID(requestContext context.Context, id string) (domain.User, error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

type TokenIssuer interface {
	Issue(userID, email string) (Token, error)
}
