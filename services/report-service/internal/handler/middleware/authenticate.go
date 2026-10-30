package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/handler/respond"
)

const (
	bearerScheme             = "Bearer"
	authorizationHeaderParts = 2
)

type TokenVerifier interface {
	Verify(token string) (userID string, err error)
}

type Principal struct {
	UserID      string
	AccessToken string
}

type principalContextKey struct{}

func Authenticate(verifier TokenVerifier, writer respond.Writer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		token, err := bearerToken(request.Header.Get("Authorization"))

		var userID string
		if err == nil {
			userID, err = verifier.Verify(token)
		}

		if err != nil {
			writer.Error(response, request, errUnauthorized)
			return
		}

		principal := Principal{UserID: userID, AccessToken: token}
		authenticatedContext := context.WithValue(request.Context(), principalContextKey{}, principal)
		next.ServeHTTP(response, request.WithContext(authenticatedContext))
	})
}

func PrincipalFrom(requestContext context.Context) Principal {
	principal, _ := requestContext.Value(principalContextKey{}).(Principal)
	return principal
}

func bearerToken(header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != authorizationHeaderParts || !strings.EqualFold(parts[0], bearerScheme) {
		return "", errUnauthorized
	}

	return parts[1], nil
}
