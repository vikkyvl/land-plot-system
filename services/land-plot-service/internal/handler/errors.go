package handler

import "github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"

const codeInvalidRequest = "invalid_request"

var (
	errInvalidCreatePlotBody = domain.NewError(domain.KindInvalidRequest, codeInvalidRequest,
		"request body must contain a name and points")

	errInvalidUpdatePlotBody = domain.NewError(domain.KindInvalidRequest, codeInvalidRequest,
		"request body must contain valid name or points fields")
)
