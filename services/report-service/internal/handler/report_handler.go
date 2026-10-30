package handler

import (
	"net/http"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/handler/middleware"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/handler/respond"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/service"
)

type reportHandler struct {
	reports   ReportService
	responses respond.Writer
}

func (handler *reportHandler) create(response http.ResponseWriter, request *http.Request) {
	body, err := respond.DecodeJSON[createReportRequest](response, request)
	if err != nil {
		handler.responses.Error(response, request, errInvalidCreateReportBody)
		return
	}

	user := middleware.PrincipalFrom(request.Context())
	caller := service.Caller{
		UserID:      user.UserID,
		AccessToken: user.AccessToken,
		RequestID:   request.Header.Get(respond.RequestIDHeader),
	}

	report, err := handler.reports.Create(request.Context(), caller, body.PlotID)
	if err != nil {
		handler.responses.Error(response, request, err)
		return
	}

	handler.responses.JSON(response, http.StatusCreated, newReportResponse(report))
}

func (handler *reportHandler) get(response http.ResponseWriter, request *http.Request) {
	report, err := handler.reports.Get(
		request.Context(),
		middleware.PrincipalFrom(request.Context()).UserID,
		request.PathValue("id"),
	)
	if err != nil {
		handler.responses.Error(response, request, err)
		return
	}

	handler.responses.JSON(response, http.StatusOK, newReportResponse(report))
}
