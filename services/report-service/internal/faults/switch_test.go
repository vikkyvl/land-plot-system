package faults

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func serve(faultSwitch *Switch) int {
	router := http.NewServeMux()
	faultSwitch.Register(router)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, UnhealthyPath, http.NoBody))

	return response.Code
}

func TestDisabledSwitchHasNoRouteAndStaysHealthy(t *testing.T) {
	faultSwitch := NewSwitch(false, slog.New(slog.DiscardHandler))

	if code := serve(faultSwitch); code != http.StatusNotFound || !faultSwitch.Healthy() {
		t.Fatalf("status = %d, healthy = %v", code, faultSwitch.Healthy())
	}
}

func TestEnabledSwitchMarksTheServiceUnhealthyUntilARestart(t *testing.T) {
	faultSwitch := NewSwitch(true, slog.New(slog.DiscardHandler))

	if !faultSwitch.Healthy() {
		t.Fatal("a new switch is unhealthy")
	}

	if code := serve(faultSwitch); code != http.StatusNoContent || faultSwitch.Healthy() {
		t.Fatalf("status = %d, healthy = %v", code, faultSwitch.Healthy())
	}

	if restarted := NewSwitch(true, slog.New(slog.DiscardHandler)); !restarted.Healthy() {
		t.Fatal("a restarted service starts unhealthy")
	}
}

func TestGuardAnswers503OnHealthOnlyWhileUnhealthy(t *testing.T) {
	faultSwitch := NewSwitch(true, slog.New(slog.DiscardHandler))
	inner := http.HandlerFunc(
		func(response http.ResponseWriter, _ *http.Request) { response.WriteHeader(http.StatusOK) },
	)
	guarded := faultSwitch.GuardHealth(inner)

	status := func(path string) int {
		response := httptest.NewRecorder()
		guarded.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, http.NoBody))

		return response.Code
	}

	if status(healthPath) != http.StatusOK {
		t.Fatal("healthy service does not answer 200 on /health")
	}

	serve(faultSwitch)

	if status(healthPath) != http.StatusServiceUnavailable || status("/api/v1/plots") != http.StatusOK {
		t.Fatalf("health = %d, api = %d", status(healthPath), status("/api/v1/plots"))
	}
}
