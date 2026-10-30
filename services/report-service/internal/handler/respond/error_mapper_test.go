package respond

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

func writeError(t *testing.T, err error) (int, ErrorResponse, string) {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	request.Header.Set(RequestIDHeader, "request-1")

	response := httptest.NewRecorder()

	NewWriter(slog.New(slog.DiscardHandler)).Error(response, request, err)

	rawBody := response.Body.String()

	var body ErrorResponse
	if decodeErr := json.Unmarshal([]byte(rawBody), &body); decodeErr != nil {
		t.Fatalf("decode response: %v", decodeErr)
	}

	return response.Code, body, rawBody
}

func TestEveryErrorKindHasStatus(t *testing.T) {
	for kind := domain.KindInvalidRequest; kind <= domain.KindUnavailable; kind++ {
		if _, ok := statusByKind[kind]; !ok {
			t.Fatalf("statusByKind has no status for kind %d", kind)
		}
	}
}

func TestDomainErrorUsesItsKindCodeAndMessage(t *testing.T) {
	status, body, _ := writeError(t, fmt.Errorf("get report: %w", domain.ErrReportNotFound))

	if status != http.StatusNotFound || body.Code != "report_not_found" || body.RequestID != "request-1" {
		t.Fatalf("status = %d body = %#v", status, body)
	}
}

func TestUnknownErrorHidesDetails(t *testing.T) {
	status, body, rawBody := writeError(t, errors.New("database password exposed"))

	if status != http.StatusInternalServerError || body.Code != codeInternalError ||
		strings.Contains(rawBody, "password") {
		t.Fatalf("status = %d body = %s", status, rawBody)
	}
}
