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
	limiter RequestLimiter,
	allowedOrigins []string,
) http.Handler {
	router := http.NewServeMux()
	router.HandleFunc("GET /health", healthHandler(logger, serviceName))
	router.Handle("/", gateway)

	protected := sanitizeUserHeaders(limitRequests(limiter, authenticate(verifier, router)))

	return withRequestID(withCORS(allowedOrigins, protected))
}
