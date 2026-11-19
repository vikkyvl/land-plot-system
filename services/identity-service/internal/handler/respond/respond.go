package respond

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/logging"
)

const (
	RequestIDHeader = "X-Request-ID"
	maximumBodySize = 1 << 20
)

var errTrailingJSON = errors.New("request body must contain one JSON object")

type ErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

type Writer struct {
	logger *slog.Logger
}

func NewWriter(logger *slog.Logger) Writer {
	return Writer{logger: logger}
}

func (writer Writer) JSON(response http.ResponseWriter, status int, body any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)

	if err := json.NewEncoder(response).Encode(body); err != nil {
		writer.logger.Error(logWriteResponse, logging.KeyStatus, status, logging.KeyError, err)
	}
}

func DecodeJSON[Body any](response http.ResponseWriter, request *http.Request) (Body, error) {
	var body Body

	request.Body = http.MaxBytesReader(response, request.Body, maximumBodySize)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&body); err != nil {
		return body, err
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return body, errTrailingJSON
	}

	return body, nil
}
