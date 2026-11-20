package middleware

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/handler/respond"
)

const (
	bearerScheme             = "Bearer"
	authorizationHeaderParts = 2
)

type TokenVerifier interface {
	Verify(token string) (domain.Principal, error)
}

type Principal struct {
	UserID      string
	AccessToken string
	Scopes      []string
}

type principalContextKey struct{}

func Authenticate(verifier TokenVerifier, writer respond.Writer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		token, err := bearerToken(request.Header.Get("Authorization"))

		var verified domain.Principal
		if err == nil {
			verified, err = verifier.Verify(token)
		}

		if err != nil {
			response.Header().Set(headerWWWAuthenticate, rejectedRequestChallenge)
			writer.Error(response, request, errUnauthorized)

			return
		}

		principal := Principal{UserID: verified.Subject, AccessToken: token, Scopes: verified.Scopes}
		authenticatedContext := context.WithValue(request.Context(), principalContextKey{}, principal)
		next.ServeHTTP(response, request.WithContext(authenticatedContext))
	})
}

func (principal Principal) HasScope(scope string) bool {
	return slices.Contains(principal.Scopes, scope)
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
