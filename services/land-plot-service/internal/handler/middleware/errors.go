package middleware

import "github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"

var (
	errUnauthorized = domain.NewError(domain.KindUnauthorized, "unauthorized",
		"valid bearer token is required")

	errInsufficientScope = domain.NewError(domain.KindForbidden, "insufficient_scope",
		"access token lacks the required scope")
)
