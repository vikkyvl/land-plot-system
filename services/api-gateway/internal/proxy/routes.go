package proxy

import "strings"

const (
	identityService = "identity-service"
	landPlotService = "land-plot-service"
	reportService   = "report-service"
)

type route struct {
	path      string
	service   string
	exact     bool
	protected bool
}

var routes = []route{
	{path: "/api/v1/auth", service: identityService},
	{path: "/api/v1/users/profile", service: identityService, exact: true, protected: true},
	{path: "/api/v1/plots", service: landPlotService, protected: true},
	{path: "/api/v1/reports", service: reportService, protected: true},
	{path: "/api/v1/demo/chain", service: reportService, exact: true},
}

func (candidateRoute route) matches(path string) bool {
	if path == candidateRoute.path {
		return true
	}

	return !candidateRoute.exact && strings.HasPrefix(path, candidateRoute.path+"/")
}

func findRoute(path string) (route, bool) {
	for _, candidate := range routes {
		if candidate.matches(path) {
			return candidate, true
		}
	}

	return route{}, false
}

func RequiresAuthentication(path string) bool {
	matched, ok := findRoute(path)
	return ok && matched.protected
}
