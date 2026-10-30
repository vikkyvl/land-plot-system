package domain

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorIsMatchedThroughWrapping(t *testing.T) {
	err := fmt.Errorf("get report: %w", ErrReportNotFound)

	if !errors.Is(err, ErrReportNotFound) || errors.Is(err, ErrPlotNotFound) {
		t.Fatalf("errors.Is mismatch for %v", err)
	}

	var domainError *Error
	if !errors.As(err, &domainError) || domainError.Kind != KindNotFound || domainError.Code != "report_not_found" {
		t.Fatalf("errors.As = %#v", domainError)
	}
}
