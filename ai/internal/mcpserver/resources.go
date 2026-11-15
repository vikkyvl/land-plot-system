package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vleshchenko/land-plot-system/ai/internal/guard"
	"github.com/vleshchenko/land-plot-system/ai/internal/tools"
)

const (
	plotsURI           = "land://plots"
	plotURIPrefix      = "land://plots/"
	reportURIPrefix    = "land://reports/"
	plotURITemplate    = plotURIPrefix + "{plot_id}"
	reportURITemplate  = reportURIPrefix + "{report_id}"
	codeInvalidRequest = "invalid_arguments"
)

type toolError struct {
	Error struct {
		Code   string `json:"code"`
		Status int    `json:"status"`
	} `json:"error"`
}

func (handlers *service) registerResources(server *mcp.Server) {
	server.AddResource(&mcp.Resource{
		URI:         plotsURI,
		Name:        "plots",
		Title:       "Land plots",
		Description: descriptionPlotsResource,
		MIMEType:    mimeTypeJSON,
	}, handlers.readResource(func(string) (string, json.RawMessage) {
		return tools.ListPlots, json.RawMessage(`{}`)
	}))

	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: plotURITemplate,
		Name:        "plot",
		Title:       "Land plot",
		Description: descriptionPlotResource,
		MIMEType:    mimeTypeJSON,
	}, handlers.readResource(func(uri string) (string, json.RawMessage) {
		return tools.GetPlot, identifierArguments("plot_id", strings.TrimPrefix(uri, plotURIPrefix))
	}))

	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: reportURITemplate,
		Name:        "report",
		Title:       "Area report",
		Description: descriptionReportResource,
		MIMEType:    mimeTypeJSON,
	}, handlers.readResource(func(uri string) (string, json.RawMessage) {
		return tools.GetReport, identifierArguments("report_id", strings.TrimPrefix(uri, reportURIPrefix))
	}))
}

func (handlers *service) readResource(
	toolFor func(uri string) (string, json.RawMessage),
) mcp.ResourceHandler {
	return func(readContext context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		uri := request.Params.URI
		name, arguments := toolFor(uri)

		result := handlers.toolset.Call(readContext, name, arguments)
		if result.IsError {
			return nil, resourceError(uri, result.Text)
		}

		text, _ := guard.MaskInjections(result.Text)

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{URI: uri, MIMEType: mimeTypeJSON, Text: text}},
		}, nil
	}
}

func identifierArguments(field, value string) json.RawMessage {
	encoded, _ := json.Marshal(map[string]string{field: value})
	return encoded
}

func resourceError(uri, text string) error {
	var decoded toolError

	_ = json.Unmarshal([]byte(text), &decoded)

	if decoded.Error.Status == http.StatusNotFound || decoded.Error.Code == codeInvalidRequest {
		return mcp.ResourceNotFoundError(uri)
	}

	return fmt.Errorf(formatReadResource, uri, text)
}
