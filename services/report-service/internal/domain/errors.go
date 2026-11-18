package domain

import (
	"errors"
	"fmt"
)

var (
	ErrRejectedEvent   = errors.New("event rejected by the consumer")
	ErrMessageRejected = errors.New("message rejected by the broker")
)

var (
	ErrInvalidPlotID = NewError(KindInvalidRequest, "invalid_plot_id",
		"plot ID must be a valid UUID")

	ErrInvalidReportID = NewError(KindInvalidRequest, "invalid_report_id",
		"report ID must be a valid UUID")

	ErrInvalidQuery = NewError(KindInvalidRequest, "invalid_query",
		fmt.Sprintf("plot_id must be a valid UUID and limit an integer between 1 and %d", MaximumReportLimit))

	ErrPlotNotFound = NewError(KindNotFound, "plot_not_found",
		"plot not found")

	ErrReportNotFound = NewError(KindNotFound, "report_not_found",
		"report not found")

	ErrForbidden = NewError(KindForbidden, "forbidden",
		"resource belongs to another user")

	ErrInsufficientScope = NewError(KindForbidden, "insufficient_scope",
		"access token lacks the scope plots:read required to read the plot")

	ErrPlotServiceRejectedToken = NewError(KindUnauthorized, "unauthorized",
		"land-plot-service rejected the access token")

	ErrPlotDeleted = NewError(KindGone, "plot_deleted",
		"plot was deleted")

	ErrPlotNotConfirmed = NewError(KindConflict, "plot_not_confirmed",
		"plot is waiting for confirmation; retry the request later")
)

var (
	ErrElevationCountMismatch = errors.New("elevation count does not match the sampled points")
	ErrDegenerateTerrainGrid  = errors.New("sampled points do not span a plane")
)
