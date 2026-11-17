package handler

import (
	"net/http"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/handler/middleware"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/handler/respond"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/service"
)

func callerFrom(request *http.Request) service.Caller {
	principal := middleware.PrincipalFrom(request.Context())

	return service.Caller{
		UserID:      principal.UserID,
		AccessToken: principal.AccessToken,
		RequestID:   request.Header.Get(respond.RequestIDHeader),
	}
}
