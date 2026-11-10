package proxy

import "testing"

func TestFindRoute(t *testing.T) {
	tests := []struct {
		path      string
		service   string
		found     bool
		protected bool
	}{
		{path: "/api/v1/auth/login", service: identityService, found: true},
		{path: "/api/v1/users/profile", service: identityService, found: true, protected: true},
		{path: "/api/v1/users/profile/extra", found: false},
		{path: "/api/v1/plots", service: landPlotService, found: true, protected: true},
		{path: "/api/v1/plots/42", service: landPlotService, found: true, protected: true},
		{path: "/api/v1/reports/42", service: reportService, found: true, protected: true},
		{path: "/api/v1/demo/chain", service: reportService, found: true},
		{path: "/api/v1/demo/chain/extra", found: false},
		{path: "/api/v1/plot", found: false},
		{path: "/internal/plots/42", found: false},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			matched, found := findRoute(test.path)
			if matched.service != test.service || found != test.found {
				t.Fatalf(
					"findRoute(%q) = %q, %v; want %q, %v",
					test.path,
					matched.service,
					found,
					test.service,
					test.found,
				)
			}

			if got := RequiresAuthentication(test.path); got != test.protected {
				t.Fatalf("RequiresAuthentication(%q) = %v, want %v", test.path, got, test.protected)
			}
		})
	}
}
