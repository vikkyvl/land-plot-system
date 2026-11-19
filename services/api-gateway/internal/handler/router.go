package handler

import (
	"log/slog"
	"net/http"

	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/proxy"
)

func NewRouter(
	logger *slog.Logger,
	serviceName string,
	gateway http.Handler,
	verifier TokenVerifier,
	limiter RequestLimiter,
	allowedOrigins []string,
	observer RequestObserver,
) http.Handler {
	router := http.NewServeMux()
	router.HandleFunc(http.MethodGet+" "+proxy.HealthPath, healthHandler(logger, serviceName))
	router.Handle("/", gateway)

	protected := sanitizeUserHeaders(limitRequests(limiter, authenticate(verifier, router)))

	return observeRequests(observer, withRequestID(withCORS(allowedOrigins, protected)))
}
