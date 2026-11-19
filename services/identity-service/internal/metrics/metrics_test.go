package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestProfilesAndNewUsersAreCounted(t *testing.T) {
	service := New("identity-service")
	service.ProfileServed(true)
	service.ProfileServed(false)
	service.ProfileServed(false)

	want := `
		# HELP identity_profiles_served_total Profiles returned for a verified token; the web app asks for it after every sign-in.
		# TYPE identity_profiles_served_total counter
		identity_profiles_served_total{service="identity-service"} 3
		# HELP identity_users_created_total Users created just in time on their first verified token.
		# TYPE identity_users_created_total counter
		identity_users_created_total{service="identity-service"} 1
	`

	err := testutil.GatherAndCompare(
		service.registry, strings.NewReader(want), "identity_profiles_served_total", "identity_users_created_total",
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUnknownPathsShareOneRoute(t *testing.T) {
	service := New("identity-service")
	router := http.NewServeMux()
	router.HandleFunc("GET /api/v1/users/profile", func(http.ResponseWriter, *http.Request) {})
	handler := service.ObserveRequests(router)

	for _, path := range []string{"/api/v1/users/profile", "/admin", "/api/v1/users/profile/x"} {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, http.NoBody))
	}

	if count := testutil.CollectAndCount(service.requestDuration); count != 2 {
		t.Fatalf("series = %d, want the profile route and %q", count, unmatchedRoute)
	}
}
