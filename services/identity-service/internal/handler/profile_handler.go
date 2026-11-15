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
	user, err := handler.identity.Profile(request.Context(), middleware.UserID(request.Context()))
	if err != nil {
		handler.responses.Error(response, request, err)
		return
	}

	handler.responses.JSON(response, http.StatusOK, newUserResponse(user))
}
