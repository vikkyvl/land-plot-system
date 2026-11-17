package domain

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorIsMatchedThroughWrapping(t *testing.T) {
	err := fmt.Errorf("profile: %w", ErrInvalidSubject)
	otherError := NewError(KindNotFound, "other", "other error")

	if !errors.Is(err, ErrInvalidSubject) || errors.Is(err, otherError) {
		t.Fatalf("errors.Is mismatch for %v", err)
	}

	var domainError *Error
	if !errors.As(err, &domainError) || domainError.Kind != KindUnauthorized || domainError.Code != "invalid_subject" {
		t.Fatalf("errors.As = %#v", domainError)
	}
}
