package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/handler/respond"
)

const BearerScheme = "Bearer"

const authorizationHeaderParts = 2

type TokenVerifier interface {
	Verify(token string) (userID string, err error)
}

type userIDContextKey struct{}

func Authenticate(verifier TokenVerifier, writer respond.Writer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		userID, err := verifyBearerToken(verifier, request.Header.Get("Authorization"))
		if err != nil {
			writer.Error(response, request, errUnauthorized)
			return
		}

		authenticatedContext := context.WithValue(request.Context(), userIDContextKey{}, userID)
		next.ServeHTTP(response, request.WithContext(authenticatedContext))
	})
}

func UserID(requestContext context.Context) string {
	userID, _ := requestContext.Value(userIDContextKey{}).(string)
	return userID
}

func verifyBearerToken(verifier TokenVerifier, header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != authorizationHeaderParts || !strings.EqualFold(parts[0], BearerScheme) {
		return "", errUnauthorized
	}

	return verifier.Verify(parts[1])
}
