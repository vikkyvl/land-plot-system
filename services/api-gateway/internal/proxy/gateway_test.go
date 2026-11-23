package proxy

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/balancer"
	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/discovery"
)

const testGatewayID = "gateway-1"

type resolverStub struct {
	instances []discovery.Instance
	err       error
	service   string
}

func (stub *resolverStub) Resolve(_ context.Context, serviceName string) ([]discovery.Instance, error) {
	stub.service = serviceName
	return stub.instances, stub.err
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (roundTrip roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

type upstreamObservation struct {
	upstream string
	status   int
	err      error
}

type observerStub struct {
	observations []upstreamObservation
}

func (stub *observerStub) ObserveUpstream(upstream string, status int, err error, _ time.Duration) {
	stub.observations = append(stub.observations, upstreamObservation{upstream: upstream, status: status, err: err})
}

func newTestGateway(resolver ServiceResolver, transport http.RoundTripper) *Gateway {
	return newObservedTestGateway(resolver, transport, &observerStub{})
}

func newObservedTestGateway(resolver ServiceResolver, transport http.RoundTripper, observer *observerStub) *Gateway {
	gateway := NewGateway(
		resolver,
		balancer.NewRoundRobin(),
		slog.New(slog.DiscardHandler),
		time.Second,
		testGatewayID,
		observer,
	)
	gateway.proxy.Transport = observedTransport{base: transport, observer: observer}

	return gateway
}

func okResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{}`)),
	}
}

func TestGatewayRoutesRequestToResolvedInstance(t *testing.T) {
	resolver := &resolverStub{instances: []discovery.Instance{{ID: "plot-1", Address: "10.0.0.8", Port: 8082}}}
	gateway := newTestGateway(resolver, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Host != "10.0.0.8:8082" || request.URL.Path != "/api/v1/plots/plot-id" ||
			request.URL.RawQuery != "a=1" {
			t.Fatalf("upstream URL = %s", request.URL)
		}

		if request.Header.Get(HeaderGatewayInstance) != testGatewayID {
			t.Fatalf("gateway instance header = %q", request.Header.Get(HeaderGatewayInstance))
		}

		return okResponse(), nil
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/plots/plot-id?a=1", http.NoBody)
	request.Header.Set(HeaderGatewayInstance, "spoofed-gateway")

	response := httptest.NewRecorder()

	gateway.ServeHTTP(response, request)

	if response.Code != http.StatusOK || resolver.service != landPlotService {
		t.Fatalf("status = %d resolved service = %q", response.Code, resolver.service)
	}

	if servedBy := response.Header().Get(HeaderServedBy); servedBy != "plot-1" {
		t.Fatalf("X-Served-By = %q, want plot-1", servedBy)
	}
}

func TestGatewayBalancesAcrossInstances(t *testing.T) {
	resolver := &resolverStub{instances: []discovery.Instance{
		{ID: "plot-1", Address: "10.0.0.1", Port: 8082},
		{ID: "plot-2", Address: "10.0.0.2", Port: 8082},
	}}
	gateway := newTestGateway(
		resolver,
		roundTripFunc(func(*http.Request) (*http.Response, error) { return okResponse(), nil }),
	)

	servedBy := make([]string, 0, 3)

	for range 3 {
		response := httptest.NewRecorder()
		gateway.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/plots", http.NoBody))
		servedBy = append(servedBy, response.Header().Get(HeaderServedBy))
	}

	if strings.Join(servedBy, ",") != "plot-1,plot-2,plot-1" {
		t.Fatalf("served by = %v", servedBy)
	}
}

func TestGatewayErrors(t *testing.T) {
	failingTransport := roundTripFunc(
		func(*http.Request) (*http.Response, error) { return nil, errors.New("connection failed") },
	)
	oneInstance := &resolverStub{instances: []discovery.Instance{{ID: "identity-1", Address: "10.0.0.9", Port: 8081}}}

	tests := []struct {
		name       string
		resolver   ServiceResolver
		path       string
		wantStatus int
	}{
		{"unknown route", &resolverStub{}, "/unknown", http.StatusNotFound},
		{"no healthy instances", &resolverStub{}, "/api/v1/reports", http.StatusServiceUnavailable},
		{
			"discovery failure",
			&resolverStub{err: errors.New("consul down")},
			"/api/v1/reports",
			http.StatusServiceUnavailable,
		},
		{"removed login route", oneInstance, "/api/v1/auth/login", http.StatusNotFound},
		{"upstream failure", oneInstance, "/api/v1/users/profile", http.StatusBadGateway},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, http.NoBody)
			request.Header.Set(HeaderRequestID, "request-1")

			response := httptest.NewRecorder()

			newTestGateway(test.resolver, failingTransport).ServeHTTP(response, request)

			if response.Code != test.wantStatus ||
				!strings.Contains(response.Body.String(), `"request_id":"request-1"`) {
				t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestGatewayRemovesCORSHeadersFromUpstream(t *testing.T) {
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		response := okResponse()
		response.Header.Set("Access-Control-Allow-Origin", "*")
		response.Header.Set("Access-Control-Allow-Credentials", "true")
		response.Header.Set(HeaderRequestID, "request-1")
		response.Header.Set("Content-Type", "application/json")

		return response, nil
	})
	resolver := &resolverStub{instances: []discovery.Instance{{ID: "plot-1", Address: "10.0.0.1", Port: 8082}}}

	response := httptest.NewRecorder()
	newTestGateway(
		resolver,
		transport,
	).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/plots", http.NoBody))

	headers := response.Header()
	if headers.Get("Access-Control-Allow-Origin") != "" || headers.Get("Access-Control-Allow-Credentials") != "" ||
		headers.Get(HeaderRequestID) != "request-1" || headers.Get(HeaderServedBy) != "plot-1" {
		t.Fatalf("headers = %#v", headers)
	}
}

func TestGatewayObservesEveryUpstreamCall(t *testing.T) {
	resolver := &resolverStub{instances: []discovery.Instance{{ID: "report-1", Address: "10.0.0.9", Port: 8083}}}
	failure := errors.New("connection refused")
	responses := []func() (*http.Response, error){
		func() (*http.Response, error) { return okResponse(), nil },
		func() (*http.Response, error) { return nil, failure },
	}
	observer := &observerStub{}
	call := 0
	gateway := newObservedTestGateway(resolver, roundTripFunc(func(*http.Request) (*http.Response, error) {
		response, err := responses[call]()
		call++

		return response, err
	}), observer)

	for range responses {
		gateway.ServeHTTP(
			httptest.NewRecorder(),
			httptest.NewRequest(http.MethodGet, "/api/v1/reports/r-1", http.NoBody),
		)
	}

	want := []upstreamObservation{
		{upstream: "report-service", status: http.StatusOK},
		{upstream: "report-service", err: failure},
	}
	if len(observer.observations) != len(want) {
		t.Fatalf("observations = %+v, want %+v", observer.observations, want)
	}

	for index, observation := range observer.observations {
		if observation.upstream != want[index].upstream || observation.status != want[index].status ||
			!errors.Is(observation.err, want[index].err) {
			t.Fatalf("observation %d = %+v, want %+v", index, observation, want[index])
		}
	}
}
