package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/domain"
)

type IdentityService struct {
	repository UserRepository
	now        func() time.Time
}

func NewIdentityService(repository UserRepository) *IdentityService {
	return &IdentityService{repository: repository, now: time.Now}
}

func (identityService *IdentityService) Profile(
	requestContext context.Context,
	principal domain.Principal,
) (domain.User, error) {
	if _, err := uuid.Parse(principal.Subject); err != nil {
		return domain.User{}, domain.ErrInvalidSubject
	}

	user, err := identityService.repository.Upsert(
		requestContext,
		domain.NewUser(principal, identityService.now().UTC()),
	)
	if err != nil {
		return domain.User{}, fmt.Errorf(wrapSaveUser, err)
	}

	return user, nil
}
