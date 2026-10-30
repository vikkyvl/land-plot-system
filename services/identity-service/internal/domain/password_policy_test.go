package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword(strings.Repeat("p", MinimumPasswordLength)); err != nil {
		t.Fatalf("ValidatePassword() error = %v", err)
	}

	if err := ValidatePassword(strings.Repeat("p", MinimumPasswordLength-1)); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("ValidatePassword() error = %v, want ErrWeakPassword", err)
	}
}
