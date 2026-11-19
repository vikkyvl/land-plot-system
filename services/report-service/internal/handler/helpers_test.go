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
	query   service.ReportQuery
	reports []domain.Report
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

func (stub *reportServiceStub) List(
	_ context.Context,
	ownerID string,
	query service.ReportQuery,
) ([]domain.Report, error) {
	stub.ownerID, stub.query = ownerID, query

	return stub.reports, stub.err
}

func (stub *reportServiceStub) Get(_ context.Context, ownerID, _ string) (domain.Report, error) {
	stub.ownerID = ownerID
	return stub.report, stub.err
}

func (stub *reportServiceStub) Trace(_ context.Context, caller service.Caller, _, _ string) (domain.Trace, error) {
	stub.caller = caller

	return stub.trace, stub.err
}

type documentsStub struct{}

func (documentsStub) PDF(report domain.Report, language domain.Language) ([]byte, error) {
	return []byte("%PDF-1.7 " + report.PlotName + " " + string(language)), nil
}

func (documentsStub) GeoJSON(report domain.Report) ([]byte, error) {
	if len(report.Boundary) == 0 {
		return nil, domain.ErrBoundaryNotRecorded
	}

	return []byte(`{"type":"FeatureCollection","features":[]}`), nil
}

var allScopes = []string{domain.ScopePlotsRead, domain.ScopeReportsRead, domain.ScopeReportsWrite}

type verifierStub struct {
	userID string
	scopes []string
	err    error
}

func (stub verifierStub) Verify(string) (domain.Principal, error) {
	return domain.Principal{Subject: stub.userID, Scopes: stub.scopes}, stub.err
}

func newTestRouterWithVerifier(reports ReportService, verifier middleware.TokenVerifier) http.Handler {
	exports := service.NewExportService(reports, documentsStub{}, reportMetricsStub{})

	return NewRouter(slog.New(slog.DiscardHandler), testServiceName, reports, exports, verifier)
}

func newTestRouter(reports ReportService) http.Handler {
	return newTestRouterWithVerifier(reports, verifierStub{userID: testOwnerID, scopes: allScopes})
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

type reportMetricsStub struct{}

func (reportMetricsStub) ReportCreated(domain.DataStatus, domain.DataStatus) {}

func (reportMetricsStub) ReportDownloaded(string) {}
