package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/vleshchenko/land-plot-system/ai/internal/landapi"
)

const (
	testPlotID   = "4b6f1a52-38f5-4c3e-9e8b-1d2f3a4b5c6d"
	testReportID = "9a8b7c6d-5e4f-4a3b-8c2d-1e0f9a8b7c6d"
)

var errNetwork = errors.New("dial tcp 10.0.0.5:8080: connection refused")

type plotResponse struct {
	plot landapi.Plot
	err  error
}

type fakeAPI struct {
	plots         []landapi.Plot
	plotResponses []plotResponse
	report        landapi.Report
	err           error
	calls         int
	draft         landapi.PlotDraft
	changes       landapi.PlotChanges
}

func (api *fakeAPI) ListPlots(context.Context) ([]landapi.Plot, error) {
	api.calls++
	return api.plots, api.err
}

func (api *fakeAPI) GetPlot(context.Context, string) (landapi.Plot, error) {
	api.calls++

	if len(api.plotResponses) == 0 {
		return landapi.Plot{}, api.err
	}

	response := api.plotResponses[0]
	if len(api.plotResponses) > 1 {
		api.plotResponses = api.plotResponses[1:]
	}

	return response.plot, response.err
}

func (api *fakeAPI) CreatePlot(_ context.Context, draft landapi.PlotDraft) (landapi.Plot, error) {
	api.calls++
	api.draft = draft

	return landapi.Plot{ID: testPlotID, Name: draft.Name, Points: draft.Points, Version: 1, Status: "PENDING"}, api.err
}

func (api *fakeAPI) UpdatePlot(_ context.Context, plotID string, changes landapi.PlotChanges) (landapi.Plot, error) {
	api.calls++
	api.changes = changes

	return landapi.Plot{ID: plotID, Name: "Plot", Version: 2, Status: "PENDING"}, api.err
}

func (api *fakeAPI) CreateReport(context.Context, string) (landapi.Report, error) {
	api.calls++
	return api.report, api.err
}

func (api *fakeAPI) GetReport(context.Context, string) (landapi.Report, error) {
	api.calls++
	return api.report, api.err
}

func newTestToolset(api *fakeAPI) *Toolset {
	toolset := NewToolset(api, Options{MaxResultBytes: 4000}, AllNames...)
	toolset.pollInterval = time.Millisecond

	return toolset
}

func decodeResult[Value any](t *testing.T, result Result) Value {
	t.Helper()

	var value Value
	if err := json.Unmarshal([]byte(result.Text), &value); err != nil {
		t.Fatalf("result %q is not JSON: %v", result.Text, err)
	}

	return value
}

func errorOf(t *testing.T, result Result) errorDetails {
	t.Helper()

	if !result.IsError {
		t.Fatalf("result %s is not an error", result.Text)
	}

	return decodeResult[errorResult](t, result).Error
}
