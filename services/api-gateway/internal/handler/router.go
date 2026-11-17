package handler

import (
	"log/slog"
	"net/http"
)

func NewRouter(
	logger *slog.Logger,
	serviceName string,
	gateway http.Handler,
	verifier TokenVerifier,
	allowedOrigins []string,
) http.Handler {
	router := http.NewServeMux()
	router.HandleFunc("GET /health", healthHandler(logger, serviceName))
	router.Handle("/", gateway)

	return withRequestID(withCORS(allowedOrigins, sanitizeUserHeaders(authenticate(verifier, router))))
}
