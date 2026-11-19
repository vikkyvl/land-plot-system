package tools

const (
	descriptionListPlots = "List the user's land plots. Cancelled plots are not listed. " +
		"Use name_contains to find a plot by part of its name (case-insensitive)."
	descriptionGetPlot    = "Get one land plot by its ID: name, status, version, area and perimeter."
	descriptionCreatePlot = "Create a land plot from a polygon. The plot starts in status PENDING; " +
		"call wait_plot_confirmed with the returned version before creating a report. " +
		"The polygon is open: list each corner once and do not repeat the first point at the end."
	descriptionUpdatePlot = "Rename a CONFIRMED land plot or replace its polygon. Pass at least one of name or points. " +
		"The plot becomes PENDING again; call wait_plot_confirmed with the returned version. " +
		"The polygon is open: list each corner once and do not repeat the first point at the end."
	descriptionWaitPlotConfirmed = "Wait until the saga of a created or updated plot finishes. Returns outcome " +
		"confirmed (the change is saved), rolled_back (the update was compensated and the previous data restored), " +
		"cancelled (the created plot was compensated and no longer exists) or still_pending (timeout)."
	descriptionCreateReport = "Create an area report for a CONFIRMED plot. " +
		"The report contains the area in square metres, ares and hectares."
	descriptionGetReport = "Get a previously created report by its ID."

	descriptionPlotID          = "Plot ID (UUID)."
	descriptionReportID        = "Report ID (UUID)."
	descriptionName            = "Plot name, 1 to 120 characters."
	descriptionNameContains    = "Part of the plot name to search for."
	descriptionPoints          = "Polygon corners in order, 3 to 100 points, without repeating the first point."
	descriptionLatitude        = "Latitude in degrees, from -90 to 90."
	descriptionLongitude       = "Longitude in degrees, from -180 to 180."
	descriptionExpectedVersion = "Version returned by create_plot or update_plot."
	descriptionTimeoutSeconds  = "How long to wait, 1 to 40 seconds."
)
