package handler

import (
	"net/http"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/handler/respond"
)

const (
	gatewayInstanceHeader     = "X-Gateway-Instance"
	defaultTraceMessage       = "hello"
	maximumTraceMessageLength = 200
)

type traceHopResponse struct {
	Service  string `json:"service"`
	Instance string `json:"instance"`
	Protocol string `json:"protocol"`
}

type traceResponse struct {
	Message string             `json:"message"`
	Path    []traceHopResponse `json:"path"`
}

type traceHandler struct {
	reports   ReportService
	responses respond.Writer
}

func (handler *traceHandler) trace(response http.ResponseWriter, request *http.Request) {
	message := request.URL.Query().Get("message")
	if message == "" {
		message = defaultTraceMessage
	}

	if len([]rune(message)) > maximumTraceMessageLength {
		handler.responses.Error(response, request, errTraceMessageTooLong)
		return
	}

	trace, err := handler.reports.Trace(
		request.Context(),
		request.Header.Get(gatewayInstanceHeader),
		request.Header.Get(respond.RequestIDHeader),
		message,
	)
	if err != nil {
		handler.responses.Error(response, request, err)
		return
	}

	handler.responses.JSON(response, http.StatusOK, newTraceResponse(trace))
}

func newTraceResponse(trace domain.Trace) traceResponse {
	path := make([]traceHopResponse, len(trace.Path))
	for i, hop := range trace.Path {
		path[i] = traceHopResponse{Service: hop.Service, Instance: hop.Instance, Protocol: hop.Protocol}
	}

	return traceResponse{Message: trace.Message, Path: path}
}
