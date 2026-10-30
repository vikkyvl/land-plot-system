package handler

import "github.com/vleshchenko/land-plot-system/services/identity-service/internal/domain"

var errInvalidCredentialsBody = domain.NewError(domain.KindInvalidRequest, "invalid_request",
	"request body must contain valid email and password fields")
