package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

const (
	defaultPollInterval = time.Second
	unlimitedResult     = 1 << 30
)

type Options struct {
	MaxResultBytes int
}

type Toolset struct {
	api            LandAPI
	names          []string
	maxResultBytes int
	pollInterval   time.Duration
}

type handler func(toolset *Toolset, requestContext context.Context, raw json.RawMessage) Result

var handlers = map[string]handler{
	ListPlots:         (*Toolset).listPlots,
	GetPlot:           (*Toolset).getPlot,
	CreatePlot:        (*Toolset).createPlot,
	UpdatePlot:        (*Toolset).updatePlot,
	WaitPlotConfirmed: (*Toolset).waitPlotConfirmed,
	CreateReport:      (*Toolset).createReport,
	GetReport:         (*Toolset).getReport,
}

func NewToolset(api LandAPI, options Options, names ...string) *Toolset {
	allowed := make([]string, 0, len(names))
	for _, name := range names {
		if _, known := handlers[name]; known && !slices.Contains(allowed, name) {
			allowed = append(allowed, name)
		}
	}

	maxResultBytes := options.MaxResultBytes
	if maxResultBytes <= 0 {
		maxResultBytes = unlimitedResult
	}

	return &Toolset{api: api, names: allowed, maxResultBytes: maxResultBytes, pollInterval: defaultPollInterval}
}

func (toolset *Toolset) Names() []string {
	return slices.Clone(toolset.names)
}

func (toolset *Toolset) Allows(name string) bool {
	return slices.Contains(toolset.names, name)
}

func (toolset *Toolset) Definitions() []Definition {
	definitions := make([]Definition, 0, len(toolset.names))
	for _, name := range toolset.names {
		definitions = append(definitions, catalog[name])
	}

	return definitions
}

func (toolset *Toolset) Call(requestContext context.Context, name string, arguments json.RawMessage) Result {
	if !toolset.Allows(name) {
		return encodeError(errorDetails{Code: codeUnknownTool, Message: fmt.Sprintf(formatToolNotAllowed, name)})
	}

	return handlers[name](toolset, requestContext, arguments)
}

func (toolset *Toolset) listPlots(requestContext context.Context, raw json.RawMessage) Result {
	arguments, err := decodeArguments[listPlotsArguments](raw)
	if err != nil {
		return invalidArguments(err)
	}

	plots, err := toolset.api.ListPlots(requestContext)
	if err != nil {
		return apiFailure(err)
	}

	query := strings.ToLower(strings.TrimSpace(arguments.NameContains))
	result := plotListResult{Plots: []plotSummary{}}

	for _, plot := range plots {
		if strings.Contains(strings.ToLower(plot.Name), query) {
			result.Plots = append(result.Plots, summarizePlot(plot))
		}
	}

	result.Count = len(result.Plots)

	return toolset.success(result)
}

func (toolset *Toolset) getPlot(requestContext context.Context, raw json.RawMessage) Result {
	arguments, err := decodeArguments[plotIDArguments](raw)
	if err == nil {
		err = validateUUID("plot_id", arguments.PlotID)
	}

	if err != nil {
		return invalidArguments(err)
	}

	plot, err := toolset.api.GetPlot(requestContext, arguments.PlotID)
	if err != nil {
		return apiFailure(err)
	}

	return toolset.success(summarizePlot(plot))
}

func (toolset *Toolset) createPlot(requestContext context.Context, raw json.RawMessage) Result {
	arguments, err := decodeArguments[createPlotArguments](raw)
	if err != nil {
		return invalidArguments(err)
	}

	draft, err := arguments.draft()
	if err != nil {
		return invalidArguments(err)
	}

	plot, err := toolset.api.CreatePlot(requestContext, draft)
	if err != nil {
		return apiFailure(err)
	}

	return toolset.success(summarizePlot(plot))
}

func (toolset *Toolset) updatePlot(requestContext context.Context, raw json.RawMessage) Result {
	arguments, err := decodeArguments[updatePlotArguments](raw)
	if err != nil {
		return invalidArguments(err)
	}

	changes, err := arguments.changes()
	if err != nil {
		return invalidArguments(err)
	}

	plot, err := toolset.api.UpdatePlot(requestContext, arguments.PlotID, changes)
	if err != nil {
		return apiFailure(err)
	}

	return toolset.success(summarizePlot(plot))
}

func (toolset *Toolset) createReport(requestContext context.Context, raw json.RawMessage) Result {
	arguments, err := decodeArguments[plotIDArguments](raw)
	if err == nil {
		err = validateUUID("plot_id", arguments.PlotID)
	}

	if err != nil {
		return invalidArguments(err)
	}

	report, err := toolset.api.CreateReport(requestContext, arguments.PlotID)
	if err != nil {
		return apiFailure(err)
	}

	return toolset.success(summarizeReport(report))
}

func (toolset *Toolset) getReport(requestContext context.Context, raw json.RawMessage) Result {
	arguments, err := decodeArguments[reportIDArguments](raw)
	if err == nil {
		err = validateUUID("report_id", arguments.ReportID)
	}

	if err != nil {
		return invalidArguments(err)
	}

	report, err := toolset.api.GetReport(requestContext, arguments.ReportID)
	if err != nil {
		return apiFailure(err)
	}

	return toolset.success(summarizeReport(report))
}
