package tools

const (
	maximumNameLength     = 120
	minimumPolygonPoints  = 3
	maximumPolygonPoints  = 100
	minimumLatitude       = -90
	maximumLatitude       = 90
	minimumLongitude      = -180
	maximumLongitude      = 180
	minimumTimeoutSeconds = 1
	maximumTimeoutSeconds = 40
	minimumVersion        = 1
)

type Definition struct {
	Name        string
	Description string
	InputSchema map[string]any
	ReadOnly    bool
}

var catalog = map[string]Definition{
	ListPlots: {
		Name:        ListPlots,
		Description: descriptionListPlots,
		InputSchema: objectSchema(map[string]any{"name_contains": stringSchema(descriptionNameContains)}),
		ReadOnly:    true,
	},
	GetPlot: {
		Name:        GetPlot,
		Description: descriptionGetPlot,
		InputSchema: objectSchema(map[string]any{"plot_id": stringSchema(descriptionPlotID)}, "plot_id"),
		ReadOnly:    true,
	},
	CreatePlot: {
		Name:        CreatePlot,
		Description: descriptionCreatePlot,
		InputSchema: objectSchema(map[string]any{
			"name":   stringSchema(descriptionName),
			"points": pointsSchema(),
		}, "name", "points"),
	},
	UpdatePlot: {
		Name:        UpdatePlot,
		Description: descriptionUpdatePlot,
		InputSchema: objectSchema(map[string]any{
			"plot_id": stringSchema(descriptionPlotID),
			"name":    stringSchema(descriptionName),
			"points":  pointsSchema(),
		}, "plot_id"),
	},
	WaitPlotConfirmed: {
		Name:        WaitPlotConfirmed,
		Description: descriptionWaitPlotConfirmed,
		InputSchema: objectSchema(map[string]any{
			"plot_id":          stringSchema(descriptionPlotID),
			"expected_version": integerSchema(descriptionExpectedVersion, minimumVersion, nil),
			"timeout_seconds": integerSchema(
				descriptionTimeoutSeconds,
				minimumTimeoutSeconds,
				maximumTimeoutSeconds,
			),
		}, "plot_id", "expected_version", "timeout_seconds"),
		ReadOnly: true,
	},
	CreateReport: {
		Name:        CreateReport,
		Description: descriptionCreateReport,
		InputSchema: objectSchema(map[string]any{"plot_id": stringSchema(descriptionPlotID)}, "plot_id"),
	},
	GetReport: {
		Name:        GetReport,
		Description: descriptionGetReport,
		InputSchema: objectSchema(map[string]any{"report_id": stringSchema(descriptionReportID)}, "report_id"),
		ReadOnly:    true,
	},
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

func stringSchema(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func integerSchema(description string, minimum int, maximum any) map[string]any {
	schema := map[string]any{"type": "integer", "description": description, "minimum": minimum}
	if maximum != nil {
		schema["maximum"] = maximum
	}

	return schema
}

func numberSchema(description string, minimum, maximum int) map[string]any {
	return map[string]any{"type": "number", "description": description, "minimum": minimum, "maximum": maximum}
}

func pointsSchema() map[string]any {
	return map[string]any{
		"type":        "array",
		"description": descriptionPoints,
		"minItems":    minimumPolygonPoints,
		"maxItems":    maximumPolygonPoints,
		"items": objectSchema(map[string]any{
			"lat": numberSchema(descriptionLatitude, minimumLatitude, maximumLatitude),
			"lon": numberSchema(descriptionLongitude, minimumLongitude, maximumLongitude),
		}, "lat", "lon"),
	}
}
