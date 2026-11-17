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

	ErrInsufficientScope = &Error{
		Status: http.StatusForbidden, Code: "insufficient_scope",
		Message: "access token lacks the required scope",
	}

	ErrOriginNotAllowed = &Error{
		Status: http.StatusForbidden, Code: "origin_not_allowed",
		Message: "origin is not allowed to call this API",
	}

	ErrRouteNotFound = &Error{
		Status: http.StatusNotFound, Code: "route_not_found",
		Message: "route not found",
	}

	ErrRateLimited = &Error{
		Status: http.StatusTooManyRequests, Code: "rate_limited",
		Message: "too many requests; retry later",
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
