package handler

import (
	"log/slog"
	"net/http"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/handler/middleware"
	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/handler/respond"
)

func NewRouter(
	logger *slog.Logger,
	serviceName string,
	identity IdentityService,
	verifier middleware.TokenVerifier,
) http.Handler {
	responses := respond.NewWriter(logger)
	authHandler := &authHandler{identity: identity, responses: responses}
	profileHandler := &profileHandler{identity: identity, responses: responses}

	router := http.NewServeMux()
	router.HandleFunc("GET /health", healthHandler(serviceName, responses))
	router.HandleFunc("POST /api/v1/auth/register", authHandler.register)
	router.HandleFunc("POST /api/v1/auth/login", authHandler.login)
	router.Handle(
		"GET /api/v1/users/profile",
		middleware.Authenticate(verifier, responses, http.HandlerFunc(profileHandler.get)),
	)

	return router
}
