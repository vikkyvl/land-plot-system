package consul

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type resolverRoundTripFunc func(*http.Request) (*http.Response, error)

func (roundTrip resolverRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func TestResolverReturnsSortedPassingInstances(t *testing.T) {
	client := &http.Client{Transport: resolverRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/v1/health/service/land-plot-service" || request.URL.Query().Get("passing") != "true" {
			t.Fatalf("unexpected request URL: %s", request.URL.String())
		}

		body := `[
			{"Node":{"Address":"10.0.0.2"},"Service":{"ID":"plot-b","Address":"","Port":8082}},
			{"Node":{"Address":"10.0.0.1"},"Service":{"ID":"plot-a","Address":"10.0.0.3","Port":8082}}
		]`

		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}

	instances, err := NewResolver("http://consul:8500/", client).Resolve(context.Background(), "land-plot-service")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if len(instances) != 2 || instances[0].ID != "plot-a" || instances[1].Address != "10.0.0.2" {
		t.Fatalf("instances = %#v", instances)
	}
}
