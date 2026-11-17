package handler

import (
	"net/http"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/handler/middleware"
	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/handler/respond"
)

type profileHandler struct {
	identity  IdentityService
	responses respond.Writer
}

func (handler *profileHandler) get(response http.ResponseWriter, request *http.Request) {
	principal := middleware.PrincipalFrom(request.Context())

	user, err := handler.identity.Profile(request.Context(), principal)
	if err != nil {
		handler.responses.Error(response, request, err)
		return
	}

	handler.responses.JSON(response, http.StatusOK, newProfileResponse(user, principal.Scopes))
}
