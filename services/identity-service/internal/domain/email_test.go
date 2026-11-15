package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	email, err := NormalizeEmail("  Student@Example.COM ")
	if err != nil {
		t.Fatalf("NormalizeEmail() error = %v", err)
	}

	if email != "student@example.com" {
		t.Fatalf("NormalizeEmail() = %q, want student@example.com", email)
	}
}

func TestNormalizeEmailRejectsInvalidAddresses(t *testing.T) {
	for _, value := range []string{
		"Student <student@example.com>",
		"not-an-email",
		strings.Repeat("a", maximumEmailLength) + "@example.com",
	} {
		if _, err := NormalizeEmail(value); !errors.Is(err, ErrInvalidEmail) {
			t.Fatalf("NormalizeEmail(%q) error = %v, want ErrInvalidEmail", value, err)
		}
	}
}
