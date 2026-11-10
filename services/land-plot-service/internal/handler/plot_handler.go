package handler

import (
	"net/http"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/handler/middleware"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/handler/respond"
)

type plotHandler struct {
	plots     PlotService
	responses respond.Writer
}

func (handler *plotHandler) create(response http.ResponseWriter, request *http.Request) {
	body, err := respond.DecodeJSON[createPlotRequest](response, request)
	if err != nil {
		handler.responses.Error(response, request, errInvalidCreatePlotBody)
		return
	}

	plot, err := handler.plots.Create(request.Context(), domain.PlotDraft{
		OwnerID: middleware.UserID(request.Context()),
		Name:    body.Name,
		Points:  toCoordinates(body.Points),
	})
	if err != nil {
		handler.responses.Error(response, request, err)
		return
	}

	handler.responses.JSON(response, http.StatusCreated, newPlotResponse(plot))
}

func (handler *plotHandler) list(response http.ResponseWriter, request *http.Request) {
	plots, err := handler.plots.List(request.Context(), middleware.UserID(request.Context()))
	if err != nil {
		handler.responses.Error(response, request, err)
		return
	}

	plotsResponse := make([]plotResponse, 0, len(plots))
	for _, plot := range plots {
		plotsResponse = append(plotsResponse, newPlotResponse(plot))
	}

	handler.responses.JSON(response, http.StatusOK, plotsResponse)
}

func (handler *plotHandler) get(response http.ResponseWriter, request *http.Request) {
	plot, err := handler.plots.Get(request.Context(), middleware.UserID(request.Context()), request.PathValue("id"))
	if err != nil {
		handler.responses.Error(response, request, err)
		return
	}

	handler.responses.JSON(response, http.StatusOK, newPlotResponse(plot))
}

func (handler *plotHandler) update(response http.ResponseWriter, request *http.Request) {
	body, err := respond.DecodeJSON[updatePlotRequest](response, request)
	if err != nil {
		handler.responses.Error(response, request, errInvalidUpdatePlotBody)
		return
	}

	plot, err := handler.plots.Update(
		request.Context(),
		middleware.UserID(request.Context()),
		request.PathValue("id"),
		body.toChanges(),
	)
	if err != nil {
		handler.responses.Error(response, request, err)
		return
	}

	handler.responses.JSON(response, http.StatusOK, newPlotResponse(plot))
}
