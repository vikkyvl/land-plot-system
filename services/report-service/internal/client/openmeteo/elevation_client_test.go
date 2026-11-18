package openmeteo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

const testTimeout = time.Second

var testPoints = []domain.Coordinate{{Latitude: 48.16, Longitude: 24.5}, {Latitude: 50.4501234, Longitude: 30.52}}

func elevationServer(t *testing.T, status int, body string, query *string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if query != nil {
			*query = request.URL.RawQuery
		}

		response.WriteHeader(status)
		_, _ = response.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server
}

func TestElevationsSendsCoordinatesAndReadsValues(t *testing.T) {
	var query string

	server := elevationServer(t, http.StatusOK, `{"elevation":[2025.0, 169.5]}`, &query)

	elevations, err := NewElevationClient(server.URL, testTimeout).Elevations(context.Background(), testPoints)
	if err != nil || !reflect.DeepEqual(elevations, []float64{2025, 169.5}) {
		t.Fatalf("Elevations() = %v, %v", elevations, err)
	}

	want := "latitude=48.160000%2C50.450123&longitude=24.500000%2C30.520000"
	if query != want {
		t.Fatalf("query = %q, want %q", query, want)
	}
}

func TestElevationsRejectsBadResponses(t *testing.T) {
	tests := map[string]struct {
		status int
		body   string
	}{
		"error status":  {http.StatusBadRequest, `{"error":true,"reason":"Latitude must be in range of -90 to 90°."}`},
		"wrong count":   {http.StatusOK, `{"elevation":[1.0]}`},
		"missing value": {http.StatusOK, `{"elevation":[1.0, null]}`},
		"not JSON":      {http.StatusOK, `<html>`},
	}

	for title, test := range tests {
		server := elevationServer(t, test.status, test.body, nil)

		_, err := NewElevationClient(server.URL, testTimeout).Elevations(context.Background(), testPoints)
		if !errors.Is(err, ErrUnexpectedResponse) {
			t.Errorf("%s: error = %v, want ErrUnexpectedResponse", title, err)
		}
	}
}

func TestElevationsGiveUpAfterTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		response.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	started := time.Now()

	_, err := NewElevationClient(server.URL, 50*time.Millisecond).Elevations(context.Background(), testPoints)
	if err == nil || time.Since(started) > 150*time.Millisecond || strings.Contains(err.Error(), "latitude=") {
		t.Fatalf("error = %v after %v", err, time.Since(started))
	}
}

func TestElevationsRefuseMoreThanTheLimit(t *testing.T) {
	points := make([]domain.Coordinate, MaximumElevationPoints+1)

	_, err := NewElevationClient("http://127.0.0.1:1", testTimeout).Elevations(context.Background(), points)
	if !errors.Is(err, ErrTooManyElevationPoints) {
		t.Fatalf("error = %v, want ErrTooManyElevationPoints", err)
	}
}
