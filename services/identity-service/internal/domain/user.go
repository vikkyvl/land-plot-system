package domain

import (
	"strings"
	"time"
)

type User struct {
	ID        string
	Email     string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewUser(principal Principal, now time.Time) User {
	return User{
		ID:        principal.Subject,
		Email:     strings.ToLower(strings.TrimSpace(principal.Email)),
		Name:      strings.TrimSpace(principal.Name),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (user User) IsNew() bool {
	return user.CreatedAt.Equal(user.UpdatedAt)
}
