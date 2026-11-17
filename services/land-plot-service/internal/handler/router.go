package handler

import (
	"log/slog"
	"net/http"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/handler/middleware"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/handler/respond"
)

type protectedRoute struct {
	pattern string
	scopes  []string
	handle  http.HandlerFunc
}

func NewRouter(
	logger *slog.Logger,
	serviceName, instanceID string,
	plots PlotService,
	verifier middleware.TokenVerifier,
) http.Handler {
	responses := respond.NewWriter(logger)
	plotHandler := &plotHandler{plots: plots, responses: responses}
	traceHandler := &traceHandler{serviceName: serviceName, instanceID: instanceID, responses: responses}

	router := http.NewServeMux()
	router.HandleFunc("GET /health", healthHandler(serviceName, responses))

	for _, route := range protectedRoutes(plotHandler, traceHandler) {
		router.Handle(route.pattern, middleware.Authenticate(verifier, responses,
			middleware.RequireScopes(route.scopes, responses, route.handle)))
	}

	return router
}

func protectedRoutes(plots *plotHandler, trace *traceHandler) []protectedRoute {
	return []protectedRoute{
		{pattern: "POST /api/v1/plots", scopes: []string{domain.ScopePlotsWrite}, handle: plots.create},
		{pattern: "GET /api/v1/plots", scopes: []string{domain.ScopePlotsRead}, handle: plots.list},
		{pattern: "GET /api/v1/plots/{id}", scopes: []string{domain.ScopePlotsRead}, handle: plots.get},
		{pattern: "PATCH /api/v1/plots/{id}", scopes: []string{domain.ScopePlotsWrite}, handle: plots.update},
		{pattern: "GET /internal/plots/{id}", scopes: []string{domain.ScopePlotsRead}, handle: plots.get},
		{pattern: "GET /internal/demo/chain", handle: trace.trace},
	}
}
