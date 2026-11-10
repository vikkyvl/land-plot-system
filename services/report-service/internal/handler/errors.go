package handler

import (
	"fmt"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

var (
	errInvalidCreateReportBody = domain.NewError(domain.KindInvalidRequest, "invalid_request",
		"request body must contain a plot_id")

	errTraceMessageTooLong = domain.NewError(domain.KindInvalidRequest, "invalid_message",
		fmt.Sprintf("message must contain no more than %d characters", maximumTraceMessageLength))
)
