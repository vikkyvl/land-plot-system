package handler

import (
	"log/slog"
	"net/http"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/handler/middleware"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/handler/respond"
)

func NewRouter(
	logger *slog.Logger,
	serviceName, instanceID string,
	plots PlotService,
	verifier middleware.TokenVerifier,
) http.Handler {
	responses := respond.NewWriter(logger)
	plotHandler := &plotHandler{plots: plots, responses: responses}
	traceHandler := &traceHandler{serviceName: serviceName, instanceID: instanceID, responses: responses}
	protected := func(next http.HandlerFunc) http.Handler {
		return middleware.Authenticate(verifier, responses, next)
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /health", healthHandler(serviceName, responses))
	router.Handle("POST /api/v1/plots", protected(plotHandler.create))
	router.Handle("GET /api/v1/plots", protected(plotHandler.list))
	router.Handle("GET /api/v1/plots/{id}", protected(plotHandler.get))
	router.Handle("PATCH /api/v1/plots/{id}", protected(plotHandler.update))
	router.Handle("GET /internal/plots/{id}", protected(plotHandler.get))
	router.HandleFunc("GET /internal/demo/chain", traceHandler.trace)

	return router
}
