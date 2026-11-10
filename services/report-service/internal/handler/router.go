package handler

import (
	"log/slog"
	"net/http"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/handler/middleware"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/handler/respond"
)

func NewRouter(
	logger *slog.Logger,
	serviceName string,
	reports ReportService,
	verifier middleware.TokenVerifier,
) http.Handler {
	responses := respond.NewWriter(logger)
	reportHandler := &reportHandler{reports: reports, responses: responses}
	traceHandler := &traceHandler{reports: reports, responses: responses}
	protected := func(next http.HandlerFunc) http.Handler {
		return middleware.Authenticate(verifier, responses, next)
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /health", healthHandler(serviceName, responses))
	router.Handle("POST /api/v1/reports", protected(reportHandler.create))
	router.Handle("GET /api/v1/reports/{id}", protected(reportHandler.get))
	router.HandleFunc("GET /api/v1/demo/chain", traceHandler.trace)

	return router
}
