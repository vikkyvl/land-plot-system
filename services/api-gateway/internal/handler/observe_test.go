package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/proxy"
)

type requestObservation struct {
	route        string
	method       string
	status       int
	fromUpstream bool
}

type requestObserverStub struct {
	observations []requestObservation
}

func (stub *requestObserverStub) ObserveRequest(
	route, method string,
	status int,
	fromUpstream bool,
	_ time.Duration,
) {
	stub.observations = append(stub.observations, requestObservation{
		route: route, method: method, status: status, fromUpstream: fromUpstream,
	})
}

func observedRouter(upstream http.Handler, verifier TokenVerifier, observer RequestObserver) http.Handler {
	return NewRouter(
		slog.New(slog.DiscardHandler),
		"api-gateway",
		upstream,
		verifier,
		&limiterStub{allowed: true},
		testOrigins,
		observer,
	)
}

func TestRequestsAreObservedByRouteTemplate(t *testing.T) {
	upstream := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set(proxy.HeaderServedBy, "plot-1")
		response.WriteHeader(http.StatusNoContent)
	})
	verifier := verifierStub{userID: "user-1", scopes: []string{proxy.ScopePlotsWrite}}
	observer := &requestObserverStub{}
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/plots/7f3c2b9e-0d4a-4e51-9a7b-2d1c3e4f5a6b", http.NoBody)
	request.Header.Set("Authorization", "Bearer token")

	serve(observedRouter(upstream, verifier, observer), request)

	want := requestObservation{
		route: "/api/v1/plots/{id}", method: http.MethodDelete, status: http.StatusNoContent, fromUpstream: true,
	}
	if len(observer.observations) != 1 || observer.observations[0] != want {
		t.Fatalf("observations = %+v, want %+v", observer.observations, want)
	}
}

func TestRejectedRequestsAreObservedAsGatewayResponses(t *testing.T) {
	upstream := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("upstream called") })
	observer := &requestObserverStub{}

	serve(
		observedRouter(upstream, verifierStub{err: errors.New("invalid")}, observer),
		httptest.NewRequest(http.MethodGet, "/api/v1/reports/r-1/pdf", http.NoBody),
	)

	want := requestObservation{
		route: "/api/v1/reports/{id}/pdf", method: http.MethodGet, status: http.StatusUnauthorized,
	}
	if len(observer.observations) != 1 || observer.observations[0] != want {
		t.Fatalf("observations = %+v, want %+v", observer.observations, want)
	}
}

func TestUnknownPathsShareOneRouteLabel(t *testing.T) {
	observer := &requestObserverStub{}
	router := observedRouter(http.NotFoundHandler(), verifierStub{}, observer)

	for _, path := range []string{"/wp-login.php", "/api/v1/plots/a/b/c", "/api/v2/anything"} {
		serve(router, httptest.NewRequest(http.MethodGet, path, http.NoBody))
	}

	for _, observation := range observer.observations {
		if observation.route != proxy.UnmatchedRoute {
			t.Fatalf("route = %q, want %q", observation.route, proxy.UnmatchedRoute)
		}
	}
}

func TestHealthRequestsKeepTheirRoute(t *testing.T) {
	observer := &requestObserverStub{}

	serve(
		observedRouter(http.NotFoundHandler(), verifierStub{}, observer),
		httptest.NewRequest(http.MethodGet, proxy.HealthPath, http.NoBody),
	)

	if len(observer.observations) != 1 || observer.observations[0].route != proxy.HealthPath ||
		observer.observations[0].status != http.StatusOK {
		t.Fatalf("observations = %+v", observer.observations)
	}
}
