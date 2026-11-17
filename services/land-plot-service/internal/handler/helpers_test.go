package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

const (
	testServiceName = "land-plot-service"
	testInstanceID  = "plot-1"
	testOwnerID     = "owner-1"
	testPlotBody    = `{"name":"Plot","points":[{"lat":50,"lon":30},{"lat":50,"lon":30.001},{"lat":50.001,"lon":30}]}`
)

type plotServiceStub struct {
	ownerID   string
	changes   domain.PlotChanges
	deletedID string
	plot      domain.Plot
	err       error
}

func (stub *plotServiceStub) Create(_ context.Context, draft domain.PlotDraft) (domain.Plot, error) {
	stub.ownerID = draft.OwnerID
	return stub.plot, stub.err
}

func (stub *plotServiceStub) List(_ context.Context, ownerID string) ([]domain.Plot, error) {
	stub.ownerID = ownerID
	return []domain.Plot{stub.plot}, stub.err
}

func (stub *plotServiceStub) Get(_ context.Context, ownerID, _ string) (domain.Plot, error) {
	stub.ownerID = ownerID
	return stub.plot, stub.err
}

func (stub *plotServiceStub) Update(
	_ context.Context,
	ownerID, _ string,
	changes domain.PlotChanges,
) (domain.Plot, error) {
	stub.ownerID = ownerID
	stub.changes = changes

	return stub.plot, stub.err
}

func (stub *plotServiceStub) Delete(_ context.Context, ownerID, plotID string) error {
	stub.ownerID = ownerID
	stub.deletedID = plotID

	return stub.err
}

func (stub *plotServiceStub) Preview(points []domain.Coordinate) domain.PolygonInspection {
	return domain.InspectPolygon(points, domain.LocalEllipsoidCalculator{})
}

var allScopes = []string{domain.ScopePlotsRead, domain.ScopePlotsWrite}

type verifierStub struct {
	userID string
	scopes []string
	err    error
}

func (stub verifierStub) Verify(string) (domain.Principal, error) {
	return domain.Principal{Subject: stub.userID, Scopes: stub.scopes}, stub.err
}

func newTestRouter(plots PlotService) http.Handler {
	return newTestRouterWithScopes(plots, allScopes...)
}

func newTestRouterWithScopes(plots PlotService, scopes ...string) http.Handler {
	logger := slog.New(slog.DiscardHandler)
	return NewRouter(logger, testServiceName, testInstanceID, plots, verifierStub{userID: testOwnerID, scopes: scopes})
}

func authorizedRequest(method, target, body string) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer valid-token")

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
