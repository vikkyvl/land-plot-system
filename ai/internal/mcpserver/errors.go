package mcpserver

import "errors"

var (
	errInvalidPromptPlotID      = errors.New("plot_id must be a UUID")
	errInvalidPromptName        = errors.New("name must contain from 1 to 120 characters")
	errInvalidPromptCoordinates = errors.New("coordinates must contain from 1 to 2000 characters")
)
