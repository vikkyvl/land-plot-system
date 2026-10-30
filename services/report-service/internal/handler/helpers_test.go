package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/handler/middleware"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/service"
)

const (
	testServiceName = "report-service"
	testOwnerID     = "owner-1"
	testAccessToken = "access-token"
)

type reportServiceStub struct {
	caller  service.Caller
	plotID  string
	ownerID string
	report  domain.Report
	trace   domain.Trace
	err     error
}

func (stub *reportServiceStub) Create(_ context.Context, caller service.Caller, plotID string) (domain.Report, error) {
	stub.caller, stub.plotID = caller, plotID
	return stub.report, stub.err
}

func (stub *reportServiceStub) Get(_ context.Context, ownerID, _ string) (domain.Report, error) {
	stub.ownerID = ownerID
	return stub.report, stub.err
}

func (stub *reportServiceStub) Trace(context.Context, string, string, string) (domain.Trace, error) {
	return stub.trace, stub.err
}

type verifierStub struct {
	userID string
	err    error
}

func (stub verifierStub) Verify(string) (string, error) { return stub.userID, stub.err }

func newTestRouterWithVerifier(reports ReportService, verifier middleware.TokenVerifier) http.Handler {
	return NewRouter(slog.New(slog.DiscardHandler), testServiceName, reports, verifier)
}

func newTestRouter(reports ReportService) http.Handler {
	return newTestRouterWithVerifier(reports, verifierStub{userID: testOwnerID})
}

func authorizedRequest(method, target, body string) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+testAccessToken)

	return request
}

func serve(router http.Handler, request *http.Request) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	return response
}

func decodeBody[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()

	var body T
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	return body
}
