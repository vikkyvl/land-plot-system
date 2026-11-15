package middleware

import "github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"

var errUnauthorized = domain.NewError(domain.KindUnauthorized, "unauthorized",
	"valid bearer token is required")
