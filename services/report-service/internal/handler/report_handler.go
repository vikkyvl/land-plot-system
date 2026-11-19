package handler

import (
	"context"
	"net/http"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/handler/middleware"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/handler/respond"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/service"
)

const (
	queryPlotID   = "plot_id"
	queryLimit    = "limit"
	queryLanguage = "lang"
)

type reportHandler struct {
	reports   ReportService
	exports   ExportService
	responses respond.Writer
}

type exportFunc func(requestContext context.Context, ownerID, reportID, language string) (domain.ReportFile, error)

func (handler *reportHandler) pdf(response http.ResponseWriter, request *http.Request) {
	handler.export(response, request, handler.exports.PDF)
}

func (handler *reportHandler) geojson(response http.ResponseWriter, request *http.Request) {
	handler.export(response, request, handler.exports.GeoJSON)
}

func (handler *reportHandler) export(response http.ResponseWriter, request *http.Request, produce exportFunc) {
	file, err := produce(
		request.Context(),
		middleware.PrincipalFrom(request.Context()).UserID,
		request.PathValue("id"),
		request.URL.Query().Get(queryLanguage),
	)
	if err != nil {
		handler.responses.Error(response, request, err)
		return
	}

	writeFile(response, file)
}

func (handler *reportHandler) create(response http.ResponseWriter, request *http.Request) {
	body, err := respond.DecodeJSON[createReportRequest](response, request)
	if err != nil {
		handler.responses.Error(response, request, errInvalidCreateReportBody)
		return
	}

	report, err := handler.reports.Create(request.Context(), callerFrom(request), body.PlotID)
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

func (handler *reportHandler) list(response http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()

	reports, err := handler.reports.List(
		request.Context(),
		middleware.PrincipalFrom(request.Context()).UserID,
		service.ReportQuery{PlotID: query.Get(queryPlotID), Limit: query.Get(queryLimit)},
	)
	if err != nil {
		handler.responses.Error(response, request, err)
		return
	}

	reportsResponse := make([]reportResponse, 0, len(reports))
	for _, report := range reports {
		reportsResponse = append(reportsResponse, newReportResponse(report))
	}

	handler.responses.JSON(response, http.StatusOK, reportsResponse)
}
