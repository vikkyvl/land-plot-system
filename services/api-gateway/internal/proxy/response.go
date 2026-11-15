package proxy

import (
	"encoding/json"
	"net/http"
)

type errorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func WriteJSON(response http.ResponseWriter, status int, body any) error {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)

	return json.NewEncoder(response).Encode(body)
}

func WriteError(response http.ResponseWriter, request *http.Request, gatewayError *Error) {
	_ = WriteJSON(response, gatewayError.Status, errorResponse{
		Code:      gatewayError.Code,
		Message:   gatewayError.Message,
		RequestID: request.Header.Get(HeaderRequestID),
	})
}
