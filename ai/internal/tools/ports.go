package tools

import (
	"context"

	"github.com/vleshchenko/land-plot-system/ai/internal/landapi"
)

type LandAPI interface {
	ListPlots(requestContext context.Context) ([]landapi.Plot, error)
	GetPlot(requestContext context.Context, plotID string) (landapi.Plot, error)
	CreatePlot(requestContext context.Context, draft landapi.PlotDraft) (landapi.Plot, error)
	UpdatePlot(requestContext context.Context, plotID string, changes landapi.PlotChanges) (landapi.Plot, error)
	CreateReport(requestContext context.Context, plotID string) (landapi.Report, error)
	GetReport(requestContext context.Context, reportID string) (landapi.Report, error)
}
