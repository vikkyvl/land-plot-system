package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/domain"
)

type IdentityService struct {
	repository UserRepository
	hasher     PasswordHasher
	tokens     TokenIssuer
	now        func() time.Time
}

func NewIdentityService(repository UserRepository, hasher PasswordHasher, tokens TokenIssuer) *IdentityService {
	return &IdentityService{repository: repository, hasher: hasher, tokens: tokens, now: time.Now}
}

func (identityService *IdentityService) Register(
	requestContext context.Context,
	email, password string,
) (domain.User, error) {
	normalizedEmail, err := domain.NormalizeEmail(email)
	if err != nil {
		return domain.User{}, err
	}

	if err := domain.ValidatePassword(password); err != nil {
		return domain.User{}, err
	}

	passwordHash, err := identityService.hasher.Hash(password)
	if err != nil {
		return domain.User{}, fmt.Errorf(wrapHashPassword, err)
	}

	user := domain.User{
		ID:           uuid.NewString(),
		Email:        normalizedEmail,
		PasswordHash: passwordHash,
		CreatedAt:    identityService.now().UTC(),
	}
	if err := identityService.repository.Create(requestContext, user); err != nil {
		return domain.User{}, fmt.Errorf(wrapCreateUser, err)
	}

	return user, nil
}

func (identityService *IdentityService) Login(requestContext context.Context, email, password string) (Token, error) {
	normalizedEmail, err := domain.NormalizeEmail(email)
	if err != nil {
		return Token{}, domain.ErrInvalidCredentials
	}

	user, err := identityService.repository.FindByEmail(requestContext, normalizedEmail)
	if errors.Is(err, domain.ErrUserNotFound) {
		return Token{}, domain.ErrInvalidCredentials
	}

	if err != nil {
		return Token{}, fmt.Errorf(wrapFindUserByEmail, err)
	}

	if err := identityService.hasher.Compare(user.PasswordHash, password); err != nil {
		return Token{}, domain.ErrInvalidCredentials
	}

	token, err := identityService.tokens.Issue(user.ID, user.Email)
	if err != nil {
		return Token{}, fmt.Errorf(wrapIssueAccessToken, err)
	}

	return token, nil
}

func (identityService *IdentityService) Profile(requestContext context.Context, userID string) (domain.User, error) {
	user, err := identityService.repository.FindByID(requestContext, userID)
	if err != nil {
		return domain.User{}, fmt.Errorf(wrapFindUserByID, err)
	}

	return user, nil
}
