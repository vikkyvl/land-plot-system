package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/domain"
)

const testPassword = "password123"

type userRepositoryStub struct {
	created domain.User
	user    domain.User
	err     error
}

func (stub *userRepositoryStub) Create(_ context.Context, user domain.User) error {
	stub.created = user
	return stub.err
}

func (stub *userRepositoryStub) FindByEmail(context.Context, string) (domain.User, error) {
	return stub.user, stub.err
}

func (stub *userRepositoryStub) FindByID(context.Context, string) (domain.User, error) {
	return stub.user, stub.err
}

type passwordHasherStub struct {
	compareError error
}

func (passwordHasherStub) Hash(string) (string, error)       { return "password-hash", nil }
func (stub passwordHasherStub) Compare(string, string) error { return stub.compareError }

type tokenIssuerStub struct{}

func (tokenIssuerStub) Issue(string, string) (Token, error) {
	return Token{Value: "access-token"}, nil
}

func TestRegisterNormalizesAndHashesCredentials(t *testing.T) {
	repository := &userRepositoryStub{}
	identity := NewIdentityService(repository, passwordHasherStub{}, tokenIssuerStub{})
	fixedNow := time.Date(2026, time.October, 26, 12, 0, 0, 0, time.UTC)
	identity.now = func() time.Time { return fixedNow }

	user, err := identity.Register(context.Background(), " Student@Example.COM ", testPassword)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if user.Email != "student@example.com" || user.PasswordHash != "password-hash" || user.ID == "" {
		t.Fatalf("Register() user = %#v", user)
	}

	if !repository.created.CreatedAt.Equal(fixedNow) {
		t.Fatalf("Register() created at = %v", repository.created.CreatedAt)
	}
}

func TestRegisterRejectsWeakPassword(t *testing.T) {
	repository := &userRepositoryStub{}

	_, err := NewIdentityService(
		repository,
		passwordHasherStub{},
		tokenIssuerStub{},
	).Register(context.Background(), "a@example.com", "short")

	if !errors.Is(err, domain.ErrWeakPassword) || repository.created.ID != "" {
		t.Fatalf("Register() error = %v, created = %#v", err, repository.created)
	}
}

func TestLoginHidesCredentialDetails(t *testing.T) {
	tests := map[string]struct {
		repository *userRepositoryStub
		hasher     passwordHasherStub
	}{
		"unknown email": {&userRepositoryStub{err: domain.ErrUserNotFound}, passwordHasherStub{}},
		"wrong password": {
			&userRepositoryStub{user: domain.User{ID: "user-1"}},
			passwordHasherStub{compareError: errors.New("mismatch")},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := NewIdentityService(
				test.repository,
				test.hasher,
				tokenIssuerStub{},
			).Login(context.Background(), "student@example.com", testPassword)
			if !errors.Is(err, domain.ErrInvalidCredentials) {
				t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
			}
		})
	}
}

func TestLoginReturnsToken(t *testing.T) {
	repository := &userRepositoryStub{
		user: domain.User{ID: "user-1", Email: "student@example.com", PasswordHash: "hash"},
	}

	token, err := NewIdentityService(
		repository,
		passwordHasherStub{},
		tokenIssuerStub{},
	).Login(context.Background(), "student@example.com", testPassword)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if token.Value != "access-token" {
		t.Fatalf("Login() token = %#v", token)
	}
}
