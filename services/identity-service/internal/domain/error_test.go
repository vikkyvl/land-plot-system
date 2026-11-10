package domain

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorIsMatchedThroughWrapping(t *testing.T) {
	err := fmt.Errorf("create user: %w", ErrUserExists)

	if !errors.Is(err, ErrUserExists) || errors.Is(err, ErrUserNotFound) {
		t.Fatalf("errors.Is mismatch for %v", err)
	}

	var domainError *Error
	if !errors.As(err, &domainError) || domainError.Kind != KindConflict || domainError.Code != "user_exists" {
		t.Fatalf("errors.As = %#v", domainError)
	}
}
