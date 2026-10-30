package domain

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorIsMatchedThroughWrapping(t *testing.T) {
	err := fmt.Errorf("get plot: %w", ErrPlotNotFound)

	if !errors.Is(err, ErrPlotNotFound) || errors.Is(err, ErrForbidden) {
		t.Fatalf("errors.Is mismatch for %v", err)
	}

	var domainError *Error
	if !errors.As(err, &domainError) || domainError.Kind != KindNotFound || domainError.Code != "plot_not_found" {
		t.Fatalf("errors.As = %#v", domainError)
	}
}

func TestErrorMessageIsClientFacing(t *testing.T) {
	if ErrInvalidName.Error() != ErrInvalidName.Message {
		t.Fatalf("Error() = %q, want the message", ErrInvalidName.Error())
	}
}
