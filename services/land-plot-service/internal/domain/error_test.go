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

func TestErrorWithDetailsKeepsItsIdentity(t *testing.T) {
	detailed := ErrDuplicatePoint.WithDetails(DuplicatePointDetails{Indices: [2]int{0, 2}})

	if !errors.Is(fmt.Errorf("create plot: %w", detailed), ErrDuplicatePoint) || errors.Is(detailed, ErrZeroArea) {
		t.Fatalf("errors.Is mismatch for %#v", detailed)
	}

	if ErrDuplicatePoint.Details != nil {
		t.Fatalf("the shared error was changed: %#v", ErrDuplicatePoint)
	}
}
