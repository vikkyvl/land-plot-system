package handler

import (
	"log/slog"
	"net/http"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/handler/middleware"
	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/handler/respond"
)

type protectedRoute struct {
	pattern string
	scopes  []string
	handle  http.HandlerFunc
}

func NewRouter(
	logger *slog.Logger,
	serviceName string,
	identity IdentityService,
	verifier middleware.TokenVerifier,
) http.Handler {
	responses := respond.NewWriter(logger)
	profileHandler := &profileHandler{identity: identity, responses: responses}

	router := http.NewServeMux()
	router.HandleFunc("GET /health", healthHandler(serviceName, responses))

	for _, route := range protectedRoutes(profileHandler) {
		router.Handle(route.pattern, middleware.Authenticate(verifier, responses,
			middleware.RequireScopes(route.scopes, responses, route.handle)))
	}

	return router
}

func protectedRoutes(profiles *profileHandler) []protectedRoute {
	return []protectedRoute{
		{pattern: "GET /api/v1/users/profile", scopes: []string{domain.ScopeOpenID}, handle: profiles.get},
	}
}
