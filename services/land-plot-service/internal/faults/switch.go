package faults

import (
	"log/slog"
	"net/http"
	"sync/atomic"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/logging"
)

const (
	UnhealthyPath = "/debug/unhealthy"
	healthPath    = "/health"
	unhealthyBody = `{"status":"unhealthy"}`
	contentType   = "application/json"

	logMarkedUnhealthy = "marked unhealthy for a demo"
)

type Switch struct {
	enabled   bool
	unhealthy atomic.Bool
	logger    *slog.Logger
}

func NewSwitch(enabled bool, logger *slog.Logger) *Switch {
	return &Switch{enabled: enabled, logger: logger}
}

func (faultSwitch *Switch) Healthy() bool {
	return !faultSwitch.unhealthy.Load()
}

func (faultSwitch *Switch) Register(router *http.ServeMux) {
	if !faultSwitch.enabled {
		return
	}

	router.HandleFunc(http.MethodPost+" "+UnhealthyPath, func(response http.ResponseWriter, request *http.Request) {
		faultSwitch.unhealthy.Store(true)
		faultSwitch.logger.WarnContext(request.Context(), logMarkedUnhealthy, logging.KeyRoute, UnhealthyPath)
		response.WriteHeader(http.StatusNoContent)
	})
}

func (faultSwitch *Switch) GuardHealth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != healthPath || faultSwitch.Healthy() {
			next.ServeHTTP(response, request)
			return
		}

		request.Pattern = http.MethodGet + " " + healthPath

		response.Header().Set("Content-Type", contentType)
		response.WriteHeader(http.StatusServiceUnavailable)
		_, _ = response.Write([]byte(unhealthyBody))
	})
}
