package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/handler/respond"
)

const BearerScheme = "Bearer"

const authorizationHeaderParts = 2

type TokenVerifier interface {
	Verify(token string) (domain.Principal, error)
}

type principalContextKey struct{}

func Authenticate(verifier TokenVerifier, writer respond.Writer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		principal, err := verifyBearerToken(verifier, request.Header.Get("Authorization"))
		if err != nil {
			response.Header().Set(headerWWWAuthenticate, rejectedRequestChallenge)
			writer.Error(response, request, errUnauthorized)

			return
		}

		authenticatedContext := context.WithValue(request.Context(), principalContextKey{}, principal)
		next.ServeHTTP(response, request.WithContext(authenticatedContext))
	})
}

func PrincipalFrom(requestContext context.Context) domain.Principal {
	principal, _ := requestContext.Value(principalContextKey{}).(domain.Principal)
	return principal
}

func UserID(requestContext context.Context) string {
	return PrincipalFrom(requestContext).Subject
}

func verifyBearerToken(verifier TokenVerifier, header string) (domain.Principal, error) {
	parts := strings.Fields(header)
	if len(parts) != authorizationHeaderParts || !strings.EqualFold(parts[0], BearerScheme) {
		return domain.Principal{}, errUnauthorized
	}

	return verifier.Verify(parts[1])
}
