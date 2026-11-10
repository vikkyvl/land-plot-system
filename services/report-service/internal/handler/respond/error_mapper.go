package respond

import (
	"errors"
	"net/http"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/logging"
)

const (
	codeInternalError    = "internal_error"
	messageInternalError = "internal server error"
)

var statusByKind = map[domain.ErrorKind]int{
	domain.KindInvalidRequest: http.StatusBadRequest,
	domain.KindValidation:     http.StatusUnprocessableEntity,
	domain.KindUnauthorized:   http.StatusUnauthorized,
	domain.KindForbidden:      http.StatusForbidden,
	domain.KindNotFound:       http.StatusNotFound,
	domain.KindConflict:       http.StatusConflict,
	domain.KindUnavailable:    http.StatusServiceUnavailable,
}

func (writer Writer) Error(response http.ResponseWriter, request *http.Request, err error) {
	status, code, message := http.StatusInternalServerError, codeInternalError, messageInternalError

	if domainError, ok := errors.AsType[*domain.Error](err); ok {
		if kindStatus, known := statusByKind[domainError.Kind]; known {
			status, code, message = kindStatus, domainError.Code, domainError.Message
		}
	}

	if status >= http.StatusInternalServerError {
		writer.logger.Error(
			logHandleRequest,
			logging.KeyError, err,
			logging.KeyRequestID, request.Header.Get(RequestIDHeader),
		)
	}

	writer.JSON(response, status, ErrorResponse{
		Code:      code,
		Message:   message,
		RequestID: request.Header.Get(RequestIDHeader),
	})
}
