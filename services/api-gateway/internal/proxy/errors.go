package proxy

import "net/http"

type Error struct {
	Status  int
	Code    string
	Message string
}

func (gatewayError *Error) Error() string {
	return gatewayError.Message
}

var (
	ErrUnauthorized = &Error{
		Status: http.StatusUnauthorized, Code: "unauthorized",
		Message: "valid bearer token is required",
	}

	ErrRouteNotFound = &Error{
		Status: http.StatusNotFound, Code: "route_not_found",
		Message: "route not found",
	}

	ErrServiceUnavailable = &Error{
		Status: http.StatusServiceUnavailable, Code: "service_unavailable",
		Message: "service is temporarily unavailable",
	}

	ErrBadGateway = &Error{
		Status: http.StatusBadGateway, Code: "bad_gateway",
		Message: "upstream request failed",
	}
)
