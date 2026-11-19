package landapi

import (
	"context"
	"net/http"
	"net/url"
)

const (
	pathPlots   = "/api/v1/plots"
	pathReports = "/api/v1/reports"
)

func (client *Client) ListPlots(requestContext context.Context) ([]Plot, error) {
	var plots []Plot

	err := client.call(requestContext, request{method: http.MethodGet, path: pathPlots, authenticated: true}, &plots)

	return plots, err
}

func (client *Client) GetPlot(requestContext context.Context, plotID string) (Plot, error) {
	var plot Plot

	err := client.call(
		requestContext,
		request{method: http.MethodGet, path: plotPath(plotID), authenticated: true},
		&plot,
	)

	return plot, err
}

func (client *Client) CreatePlot(requestContext context.Context, draft PlotDraft) (Plot, error) {
	var plot Plot

	err := client.call(
		requestContext,
		request{method: http.MethodPost, path: pathPlots, body: draft, authenticated: true},
		&plot,
	)

	return plot, err
}

func (client *Client) UpdatePlot(requestContext context.Context, plotID string, changes PlotChanges) (Plot, error) {
	var plot Plot

	err := client.call(
		requestContext,
		request{method: http.MethodPatch, path: plotPath(plotID), body: changes, authenticated: true},
		&plot,
	)

	return plot, err
}

func (client *Client) CreateReport(requestContext context.Context, plotID string) (Report, error) {
	var report Report

	err := client.call(
		requestContext,
		request{
			method:        http.MethodPost,
			path:          pathReports,
			body:          createReportRequest{PlotID: plotID},
			authenticated: true,
		},
		&report,
	)

	return report, err
}

func (client *Client) GetReport(requestContext context.Context, reportID string) (Report, error) {
	var report Report

	err := client.call(
		requestContext,
		request{method: http.MethodGet, path: pathReports + "/" + url.PathEscape(reportID), authenticated: true},
		&report,
	)

	return report, err
}

func plotPath(plotID string) string {
	return pathPlots + "/" + url.PathEscape(plotID)
}
