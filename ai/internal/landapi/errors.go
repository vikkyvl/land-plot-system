package landapi

import (
	"errors"
	"fmt"
	"net/http"
)

const (
	CodeLoginRejected    = "invalid_credentials"
	CodeUserExists       = "user_exists"
	CodePlotNotFound     = "plot_not_found"
	CodePlotNotConfirmed = "plot_not_confirmed"
)

var errMissingAccessToken = errors.New("login response has no access token")

type APIError struct {
	Status  int
	Code    string
	Message string
}

func (apiError *APIError) Error() string {
	return fmt.Sprintf(formatAPIError, apiError.Status, apiError.Code, apiError.Message)
}

func IsNotFound(err error) bool {
	apiError, ok := errors.AsType[*APIError](err)
	return ok && apiError.Status == http.StatusNotFound
}

func hasCode(err error, code string) bool {
	apiError, ok := errors.AsType[*APIError](err)
	return ok && apiError.Code == code
}

func isUnauthorized(err error) bool {
	apiError, ok := errors.AsType[*APIError](err)
	return ok && apiError.Status == http.StatusUnauthorized
}
