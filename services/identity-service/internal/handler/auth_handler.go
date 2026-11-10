package handler

import (
	"net/http"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/handler/respond"
)

type authHandler struct {
	identity  IdentityService
	responses respond.Writer
}

func (handler *authHandler) register(response http.ResponseWriter, request *http.Request) {
	credentials, err := respond.DecodeJSON[credentialsRequest](response, request)
	if err != nil {
		handler.responses.Error(response, request, errInvalidCredentialsBody)
		return
	}

	user, err := handler.identity.Register(request.Context(), credentials.Email, credentials.Password)
	if err != nil {
		handler.responses.Error(response, request, err)
		return
	}

	handler.responses.JSON(response, http.StatusCreated, newUserResponse(user))
}

func (handler *authHandler) login(response http.ResponseWriter, request *http.Request) {
	credentials, err := respond.DecodeJSON[credentialsRequest](response, request)
	if err != nil {
		handler.responses.Error(response, request, errInvalidCredentialsBody)
		return
	}

	token, err := handler.identity.Login(request.Context(), credentials.Email, credentials.Password)
	if err != nil {
		handler.responses.Error(response, request, err)
		return
	}

	handler.responses.JSON(response, http.StatusOK, newLoginResponse(token))
}
