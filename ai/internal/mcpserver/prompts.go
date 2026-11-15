package mcpserver

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vleshchenko/land-plot-system/ai/internal/guard"
)

const (
	promptAnalyzePlot          = "analyze_plot"
	promptCreateFromCoordinate = "create_plot_from_coordinates"
	maximumPromptNameLength    = 120
	maximumCoordinatesLength   = 2000
)

func (handlers *service) registerPrompts(server *mcp.Server) {
	server.AddPrompt(&mcp.Prompt{
		Name:        promptAnalyzePlot,
		Title:       "Analyze a land plot",
		Description: descriptionAnalyzePrompt,
		Arguments: []*mcp.PromptArgument{
			{Name: "plot_id", Description: descriptionPlotIDArgument, Required: true},
		},
	}, analyzePlotPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        promptCreateFromCoordinate,
		Title:       "Create a plot from coordinates",
		Description: descriptionCreatePrompt,
		Arguments: []*mcp.PromptArgument{
			{Name: "name", Description: descriptionNameArgument, Required: true},
			{Name: "coordinates", Description: descriptionCoordinatesArgument, Required: true},
		},
	}, createPlotPrompt)
}

func analyzePlotPrompt(_ context.Context, request *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	plotID := strings.TrimSpace(request.Params.Arguments["plot_id"])
	if _, err := uuid.Parse(plotID); err != nil {
		return nil, errInvalidPromptPlotID
	}

	return promptResult(descriptionAnalyzePrompt, fmt.Sprintf(textAnalyzePlot, plotID, plotID)), nil
}

func createPlotPrompt(_ context.Context, request *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	name := strings.TrimSpace(request.Params.Arguments["name"])
	coordinates := strings.TrimSpace(request.Params.Arguments["coordinates"])

	switch {
	case name == "" || utf8.RuneCountInString(name) > maximumPromptNameLength:
		return nil, errInvalidPromptName
	case coordinates == "" || utf8.RuneCountInString(coordinates) > maximumCoordinatesLength:
		return nil, errInvalidPromptCoordinates
	case guard.ContainsInjection(name) || guard.ContainsInjection(coordinates):
		return nil, guard.ErrSuspiciousInput
	}

	return promptResult(descriptionCreatePrompt, fmt.Sprintf(textCreatePlot, name, coordinates)), nil
}

func promptResult(description, text string) *mcp.GetPromptResult {
	return &mcp.GetPromptResult{
		Description: description,
		Messages:    []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: text}}},
	}
}
