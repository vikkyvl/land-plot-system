package handler

import (
	"log/slog"
	"net/http"

	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/logging"
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

	logged := logging.AccessLog(logger, routeTemplate, withCORS(allowedOrigins, protected))

	return withTraceID(observeRequests(observer, withRequestID(logged)))
}

func routeTemplate(request *http.Request) string {
	return proxy.RouteTemplate(request.URL.Path)
}
