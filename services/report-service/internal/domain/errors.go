package domain

var (
	ErrInvalidPlotID = NewError(KindInvalidRequest, "invalid_plot_id",
		"plot ID must be a valid UUID")

	ErrInvalidReportID = NewError(KindInvalidRequest, "invalid_report_id",
		"report ID must be a valid UUID")

	ErrPlotNotFound = NewError(KindNotFound, "plot_not_found",
		"plot not found")

	ErrReportNotFound = NewError(KindNotFound, "report_not_found",
		"report not found")

	ErrForbidden = NewError(KindForbidden, "forbidden",
		"resource belongs to another user")
)
