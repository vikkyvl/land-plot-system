package handler

import (
	"log/slog"
	"net/http"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/handler/middleware"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/handler/respond"
)

type protectedRoute struct {
	pattern string
	scopes  []string
	handle  http.HandlerFunc
}

func NewRouter(
	logger *slog.Logger,
	serviceName string,
	reports ReportService,
	verifier middleware.TokenVerifier,
) http.Handler {
	responses := respond.NewWriter(logger)
	reportHandler := &reportHandler{reports: reports, responses: responses}
	traceHandler := &traceHandler{reports: reports, responses: responses}

	router := http.NewServeMux()
	router.HandleFunc("GET /health", healthHandler(serviceName, responses))

	for _, route := range protectedRoutes(reportHandler, traceHandler) {
		router.Handle(route.pattern, middleware.Authenticate(verifier, responses,
			middleware.RequireScopes(route.scopes, responses, route.handle)))
	}

	return router
}

func protectedRoutes(reports *reportHandler, trace *traceHandler) []protectedRoute {
	return []protectedRoute{
		{
			pattern: "POST /api/v1/reports",
			scopes:  []string{domain.ScopeReportsWrite, domain.ScopePlotsRead},
			handle:  reports.create,
		},
		{pattern: "GET /api/v1/reports/{id}", scopes: []string{domain.ScopeReportsRead}, handle: reports.get},
		{pattern: "GET /api/v1/demo/chain", handle: trace.trace},
	}
}
