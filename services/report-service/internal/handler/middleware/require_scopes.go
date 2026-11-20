package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/handler/respond"
)

func RequireScopes(scopes []string, writer respond.Writer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		principal := PrincipalFrom(request.Context())

		for _, scope := range scopes {
			if !principal.HasScope(scope) {
				response.Header().Set(headerWWWAuthenticate,
					fmt.Sprintf(formatInsufficientScope, strings.Join(scopes, scopeSeparator)))
				writer.Error(response, request, errInsufficientScope)

				return
			}
		}

		next.ServeHTTP(response, request)
	})
}
