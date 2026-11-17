package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/domain"
)

const testSubject = "164d085d-f431-498a-819a-e5a1ec00e1c1"

var errDatabase = errors.New("database is down")

type userRepositoryStub struct {
	saved []domain.User
	err   error
}

func (stub *userRepositoryStub) Upsert(_ context.Context, user domain.User) (domain.User, error) {
	stub.saved = append(stub.saved, user)
	return user, stub.err
}

func newTestService(repository UserRepository, now time.Time) *IdentityService {
	identity := NewIdentityService(repository)
	identity.now = func() time.Time { return now }

	return identity
}

func TestProfileSavesTheUserFromTheToken(t *testing.T) {
	repository := &userRepositoryStub{}
	now := time.Date(2026, time.November, 17, 12, 0, 0, 0, time.UTC)

	user, err := newTestService(repository, now).Profile(context.Background(), domain.Principal{
		Subject: testSubject,
		Email:   " Taras.Melnyk@Example.COM ",
		Name:    " Тарас Мельник ",
		Scopes:  []string{"openid"},
	})
	if err != nil {
		t.Fatalf("Profile() error = %v", err)
	}

	want := domain.User{
		ID:        testSubject,
		Email:     "taras.melnyk@example.com",
		Name:      "Тарас Мельник",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if user != want || len(repository.saved) != 1 || repository.saved[0] != want {
		t.Fatalf("Profile() = %#v, saved = %#v", user, repository.saved)
	}
}

func TestProfileAcceptsAServiceAccountWithoutEmail(t *testing.T) {
	repository := &userRepositoryStub{}

	user, err := newTestService(
		repository,
		time.Now(),
	).Profile(context.Background(), domain.Principal{Subject: testSubject})
	if err != nil || user.Email != "" || user.Name != "" {
		t.Fatalf("Profile() = %#v, %v", user, err)
	}
}

func TestProfileRejectsASubjectThatIsNotAUUID(t *testing.T) {
	repository := &userRepositoryStub{}

	_, err := newTestService(repository, time.Now()).Profile(context.Background(), domain.Principal{Subject: "admin"})
	if !errors.Is(err, domain.ErrInvalidSubject) || len(repository.saved) != 0 {
		t.Fatalf("Profile() error = %v, saved = %d", err, len(repository.saved))
	}
}

func TestProfileWrapsRepositoryErrors(t *testing.T) {
	_, err := newTestService(&userRepositoryStub{err: errDatabase}, time.Now()).
		Profile(context.Background(), domain.Principal{Subject: testSubject})
	if !errors.Is(err, errDatabase) {
		t.Fatalf("Profile() error = %v", err)
	}
}
