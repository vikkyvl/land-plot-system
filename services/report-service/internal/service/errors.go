package service

import (
	"errors"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

var ErrPlotServiceUnavailable = domain.NewError(domain.KindUnavailable, "plot_service_unavailable",
	"land plot service is temporarily unavailable")

var ErrInvalidPlotResponse = errors.New("land plot service returned an invalid response")
