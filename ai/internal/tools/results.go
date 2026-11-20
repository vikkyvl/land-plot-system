package tools

import (
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/vleshchenko/land-plot-system/ai/internal/landapi"
)

type Result struct {
	Text    string
	IsError bool
}

type plotSummary struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Status     string  `json:"status"`
	Version    int64   `json:"version"`
	AreaM2     float64 `json:"area_m2"`
	PerimeterM float64 `json:"perimeter_m"`
}

type reportSummary struct {
	ID           string  `json:"id"`
	PlotID       string  `json:"plot_id"`
	PlotName     string  `json:"plot_name"`
	PlotVersion  int64   `json:"plot_version"`
	AreaM2       float64 `json:"area_m2"`
	AreaHectares float64 `json:"area_hectares"`
	PerimeterM   float64 `json:"perimeter_m"`
}

type plotListResult struct {
	Count int           `json:"count"`
	Plots []plotSummary `json:"plots"`
}

type waitResult struct {
	Outcome string       `json:"outcome"`
	Plot    *plotSummary `json:"plot,omitempty"`
}

type errorDetails struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status,omitempty"`
}

type errorResult struct {
	Error errorDetails `json:"error"`
}

type truncatedResult struct {
	Truncated bool   `json:"truncated"`
	Note      string `json:"note"`
	Partial   string `json:"partial"`
}

func summarizePlot(plot landapi.Plot) plotSummary {
	return plotSummary{
		ID:         plot.ID,
		Name:       plot.Name,
		Status:     plot.Status,
		Version:    plot.Version,
		AreaM2:     plot.AreaM2,
		PerimeterM: plot.PerimeterM,
	}
}

func summarizeReport(report landapi.Report) reportSummary {
	return reportSummary{
		ID:           report.ID,
		PlotID:       report.PlotID,
		PlotName:     report.PlotName,
		PlotVersion:  report.PlotVersion,
		AreaM2:       report.AreaM2,
		AreaHectares: report.AreaHectares,
		PerimeterM:   report.PerimeterM,
	}
}

func invalidArguments(err error) Result {
	return encodeError(errorDetails{Code: codeInvalidArguments, Message: err.Error()})
}

func apiFailure(err error) Result {
	if apiError, ok := errors.AsType[*landapi.APIError](err); ok {
		return encodeError(errorDetails{Code: apiError.Code, Message: apiError.Message, Status: apiError.Status})
	}

	return encodeError(errorDetails{Code: codeGatewayUnavailable, Message: messageGatewayUnavailable})
}

func encodeError(details errorDetails) Result {
	encoded, _ := json.Marshal(errorResult{Error: details})
	return Result{Text: string(encoded), IsError: true}
}

func (toolset *Toolset) success(value any) Result {
	encoded, err := json.Marshal(value)
	if err != nil {
		return encodeError(errorDetails{Code: codeGatewayUnavailable, Message: messageGatewayUnavailable})
	}

	return Result{Text: toolset.limit(string(encoded))}
}

func (toolset *Toolset) limit(text string) string {
	if len(text) <= toolset.maxResultBytes {
		return text
	}

	partial := text
	for {
		encoded, _ := json.Marshal(truncatedResult{Truncated: true, Note: messageTruncated, Partial: partial})
		if len(encoded) <= toolset.maxResultBytes || partial == "" {
			return string(encoded)
		}

		partial = cutRunes(partial, min(len(partial)*toolset.maxResultBytes/len(encoded), len(partial)-1))
	}
}

func cutRunes(text string, length int) string {
	end := max(length, 0)
	for end > 0 && !utf8.RuneStart(text[end]) {
		end--
	}

	return strings.ToValidUTF8(text[:end], "")
}
