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

func newTestGateway(resolver ServiceResolver, transport http.RoundTripper) *Gateway {
	gateway := NewGateway(
		resolver,
		balancer.NewRoundRobin(),
		slog.New(slog.DiscardHandler),
		time.Second,
		testGatewayID,
	)
	gateway.proxy.Transport = transport

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
		{"upstream failure", oneInstance, "/api/v1/auth/login", http.StatusBadGateway},
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
