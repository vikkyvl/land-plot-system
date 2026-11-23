package httpclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/discovery"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/service"
)

const (
	testServiceName = "land-plot-service"
	testPlotBody    = `{"id":"plot-id","owner_id":"owner-1","name":"Plot","area_m2":10000,"perimeter_m":400,` +
		`"version":2,"status":"CONFIRMED","points":[{"lat":50.45,"lon":30.52},{"lat":50.45,"lon":30.5214},` +
		`{"lat":50.4509,"lon":30.5214}],"accuracy":{"area_m2_margin":520,"area_hectares_margin":0.052,` +
		`"percent":2.1,"confidence":0.95,"contributions":{"scale":0.4,"obstructions":0.6},"worst_side":1,` +
		`"ground_shift_m":4,"basis":"capture"}}`
)

var testCaller = service.Caller{UserID: "owner-1", AccessToken: "access-token", RequestID: "request-1"}

type resolverStub struct {
	instances []discovery.Instance
	err       error
}

func (stub resolverStub) Resolve(context.Context, string) ([]discovery.Instance, error) {
	return stub.instances, stub.err
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (roundTrip roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func singleInstance() resolverStub {
	return resolverStub{instances: []discovery.Instance{{ID: "plot-1", Address: "10.0.0.1", Port: 8082}}}
}

func respondWith(status int, body string) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
}

func TestPlotClientGetsProtectedPlot(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != "http://10.0.0.1:8082/internal/plots/plot-id" {
			t.Fatalf("URL = %s", request.URL.String())
		}

		if request.Header.Get("Authorization") != "Bearer access-token" ||
			request.Header.Get(requestIDHeader) != "request-1" {
			t.Fatalf("headers = %#v", request.Header)
		}

		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(testPlotBody))}, nil
	})}

	plot, err := NewPlotClient(
		singleInstance(),
		client,
		testServiceName,
	).Get(context.Background(), testCaller, "plot-id")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if plot.OwnerID != "owner-1" || plot.AreaM2 != 10_000 || plot.Version != 2 || len(plot.Points) != 3 ||
		plot.Points[1] != (domain.Coordinate{Latitude: 50.45, Longitude: 30.5214}) {
		t.Fatalf("plot = %#v", plot)
	}

	accuracy := plot.Accuracy
	if accuracy == nil || accuracy.AreaMarginM2 != 520 || accuracy.ObstructionShare != 0.6 ||
		accuracy.WorstSide == nil || *accuracy.WorstSide != 1 || accuracy.Basis != "capture" {
		t.Fatalf("accuracy = %#v", accuracy)
	}
}

func TestPlotClientRejectsPlotWithoutBoundary(t *testing.T) {
	body := `{"id":"plot-id","owner_id":"owner-1","name":"Plot","area_m2":10000,"perimeter_m":400,` +
		`"version":2,"status":"CONFIRMED","points":[]}`
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}

	_, err := NewPlotClient(singleInstance(), client, testServiceName).Get(context.Background(), testCaller, "plot-id")
	if !errors.Is(err, service.ErrInvalidPlotResponse) {
		t.Fatalf("Get() error = %v, want ErrInvalidPlotResponse", err)
	}
}

func TestPlotClientRoundRobinsResolvedInstances(t *testing.T) {
	resolver := resolverStub{instances: []discovery.Instance{
		{ID: "plot-1", Address: "10.0.0.1", Port: 8082},
		{ID: "plot-2", Address: "10.0.0.2", Port: 8082},
	}}
	hosts := make([]string, 0, 2)
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		hosts = append(hosts, request.URL.Host)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(testPlotBody))}, nil
	})}
	plotClient := NewPlotClient(resolver, client, testServiceName)

	for range 2 {
		if _, err := plotClient.Get(context.Background(), testCaller, "plot-id"); err != nil {
			t.Fatalf("Get() error = %v", err)
		}
	}

	if len(hosts) != 2 || hosts[0] != "10.0.0.1:8082" || hosts[1] != "10.0.0.2:8082" {
		t.Fatalf("hosts = %#v", hosts)
	}
}

func TestPlotClientMapsErrors(t *testing.T) {
	tests := []struct {
		name     string
		resolver resolverStub
		client   *http.Client
		want     error
	}{
		{"no healthy instances", resolverStub{}, &http.Client{}, service.ErrPlotServiceUnavailable},
		{"forbidden", singleInstance(), respondWith(http.StatusForbidden, `{"code":"forbidden"}`), domain.ErrForbidden},
		{
			"insufficient scope",
			singleInstance(),
			respondWith(
				http.StatusForbidden,
				`{"code":"insufficient_scope","message":"access token lacks the required scope"}`,
			),
			domain.ErrInsufficientScope,
		},
		{
			"rejected token",
			singleInstance(),
			respondWith(http.StatusUnauthorized, ""),
			domain.ErrPlotServiceRejectedToken,
		},
		{"not found", singleInstance(), respondWith(http.StatusNotFound, ""), domain.ErrPlotNotFound},
		{"deleted", singleInstance(), respondWith(http.StatusGone, `{"code":"plot_deleted"}`), domain.ErrPlotDeleted},
		{"server error", singleInstance(), respondWith(http.StatusBadGateway, ""), service.ErrPlotServiceUnavailable},
		{"malformed body", singleInstance(), respondWith(http.StatusOK, "{"), service.ErrInvalidPlotResponse},
		{
			"other plot",
			singleInstance(),
			respondWith(http.StatusOK, strings.Replace(testPlotBody, "plot-id", "other", 1)),
			service.ErrInvalidPlotResponse,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewPlotClient(
				test.resolver,
				test.client,
				testServiceName,
			).Get(context.Background(), testCaller, "plot-id")
			if !errors.Is(err, test.want) {
				t.Fatalf("Get() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestPlotClientTrace(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("Authorization") != "Bearer access-token" ||
			request.Header.Get(requestIDHeader) != "request-1" {
			t.Fatalf("headers = %#v", request.Header)
		}

		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
			`{"message":"hi","path":[{"service":"land-plot-service","instance":"plot-1","protocol":"HTTP"}]}`,
		))}, nil
	})}

	hops, err := NewPlotClient(singleInstance(), client, testServiceName).Trace(context.Background(), testCaller, "hi")
	if err != nil {
		t.Fatalf("Trace() error = %v", err)
	}

	if len(hops) != 1 || hops[0].Instance != "plot-1" {
		t.Fatalf("hops = %#v", hops)
	}
}
