package handler

import (
	"net/http"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/handler/respond"
)

const healthStatusOK = "ok"

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func healthHandler(serviceName string, responses respond.Writer) http.HandlerFunc {
	return func(response http.ResponseWriter, _ *http.Request) {
		responses.JSON(response, http.StatusOK, healthResponse{Status: healthStatusOK, Service: serviceName})
	}
}
