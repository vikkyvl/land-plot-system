package landapi

import (
	"errors"
	"fmt"
	"net/http"
)

const (
	CodePlotNotFound     = "plot_not_found"
	CodePlotNotConfirmed = "plot_not_confirmed"
)

var errMissingAccessToken = errors.New("token response has no access token")

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

func isUnauthorized(err error) bool {
	apiError, ok := errors.AsType[*APIError](err)
	return ok && apiError.Status == http.StatusUnauthorized
}
