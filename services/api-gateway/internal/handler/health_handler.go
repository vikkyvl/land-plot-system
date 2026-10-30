package handler

import (
	"log/slog"
	"net/http"

	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/logging"
	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/proxy"
)

const healthStatusOK = "ok"

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func healthHandler(logger *slog.Logger, serviceName string) http.HandlerFunc {
	return func(response http.ResponseWriter, _ *http.Request) {
		if err := proxy.WriteJSON(
			response,
			http.StatusOK,
			healthResponse{Status: healthStatusOK, Service: serviceName},
		); err != nil {
			logger.Error(logWriteHealthResponse, logging.KeyError, err)
		}
	}
}
