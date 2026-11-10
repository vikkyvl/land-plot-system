package handler

import (
	"net/http"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/handler/respond"
)

const (
	defaultTraceMessage = "hello"
	traceProtocol       = "HTTP"
)

type traceHop struct {
	Service  string `json:"service"`
	Instance string `json:"instance"`
	Protocol string `json:"protocol"`
}

type traceResponse struct {
	Message string     `json:"message"`
	Path    []traceHop `json:"path"`
}

type traceHandler struct {
	serviceName string
	instanceID  string
	responses   respond.Writer
}

func (handler *traceHandler) trace(response http.ResponseWriter, request *http.Request) {
	message := request.URL.Query().Get("message")
	if message == "" {
		message = defaultTraceMessage
	}

	handler.responses.JSON(response, http.StatusOK, traceResponse{
		Message: message,
		Path:    []traceHop{{Service: handler.serviceName, Instance: handler.instanceID, Protocol: traceProtocol}},
	})
}
