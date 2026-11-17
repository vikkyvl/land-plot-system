package proxy

import (
	"net/http"
	"slices"
	"testing"
)

func TestRequiredScopes(t *testing.T) {
	tests := []struct {
		method string
		path   string
		scopes []string
		known  bool
	}{
		{http.MethodGet, "/api/v1/plots", []string{ScopePlotsRead}, true},
		{http.MethodGet, "/api/v1/plots/plot-1", []string{ScopePlotsRead}, true},
		{http.MethodPost, "/api/v1/plots", []string{ScopePlotsWrite}, true},
		{http.MethodPatch, "/api/v1/plots/plot-1", []string{ScopePlotsWrite}, true},
		{http.MethodGet, "/api/v1/reports/report-1", []string{ScopeReportsRead}, true},
		{http.MethodPost, "/api/v1/reports", []string{ScopeReportsWrite, ScopePlotsRead}, true},
		{http.MethodGet, "/api/v1/users/profile", []string{ScopeOpenID}, true},
		{http.MethodGet, "/api/v1/demo/chain", nil, true},
		{http.MethodPost, "/api/v1/plots/preview", nil, true},
		{http.MethodGet, "/api/v1/plots/preview", []string{ScopePlotsRead}, true},
		{http.MethodDelete, "/api/v1/plots/plot-1", nil, false},
		{http.MethodGet, "/api/v1/plots/plot-1/history", nil, false},
		{http.MethodGet, "/api/v1/plots/", nil, false},
		{http.MethodGet, "/api/v1/reports", nil, false},
	}
	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			scopes, known := RequiredScopes(test.method, test.path)
			if known != test.known || !slices.Equal(scopes, test.scopes) {
				t.Fatalf("RequiredScopes() = %v, %v; want %v, %v", scopes, known, test.scopes, test.known)
			}
		})
	}
}

func TestPrincipalHasScopes(t *testing.T) {
	principal := Principal{Scopes: []string{ScopePlotsRead, ScopeReportsWrite}}

	if !principal.HasScopes([]string{ScopeReportsWrite, ScopePlotsRead}) || !principal.HasScopes(nil) ||
		principal.HasScopes([]string{ScopePlotsWrite}) {
		t.Fatal("HasScopes() is wrong")
	}
}

func TestOnlyPreviewIsPublicAndRateLimited(t *testing.T) {
	tests := []struct {
		method string
		path   string
		public bool
	}{
		{http.MethodPost, "/api/v1/plots/preview", true},
		{http.MethodGet, "/api/v1/plots/preview", false},
		{http.MethodPost, "/api/v1/plots", false},
		{http.MethodGet, "/api/v1/demo/chain", false},
		{http.MethodPost, "/api/v1/plots/preview/extra", false},
	}
	for _, test := range tests {
		if IsPublicRoute(test.method, test.path) != test.public ||
			IsRateLimitedRoute(test.method, test.path) != test.public {
			t.Fatalf("%s %s: public = %v, rate limited = %v, want %v", test.method, test.path,
				IsPublicRoute(test.method, test.path), IsRateLimitedRoute(test.method, test.path), test.public)
		}
	}
}
