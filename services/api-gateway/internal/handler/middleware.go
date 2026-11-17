package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/proxy"
)

const (
	HeaderUserID             = "X-User-ID"
	HeaderUserEmail          = "X-User-Email"
	userHeaderPrefix         = "x-user-"
	bearerScheme             = "Bearer"
	requestIDBytes           = 16
	authorizationHeaderParts = 2
	rejectedRequestChallenge = `Bearer error="invalid_token"`
	formatInsufficientScope  = `Bearer error="insufficient_scope", scope="%s"`
	scopeSeparator           = " "
)

func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestID := request.Header.Get(proxy.HeaderRequestID)
		if requestID == "" {
			requestID = newRequestID()
			request.Header.Set(proxy.HeaderRequestID, requestID)
		}

		response.Header().Set(proxy.HeaderRequestID, requestID)
		next.ServeHTTP(response, request)
	})
}

func sanitizeUserHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		for name := range request.Header {
			if strings.HasPrefix(strings.ToLower(name), userHeaderPrefix) {
				delete(request.Header, name)
			}
		}

		next.ServeHTTP(response, request)
	})
}

func authenticate(verifier TokenVerifier, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if !proxy.RequiresAuthentication(request.URL.Path) || proxy.IsPublicRoute(request.Method, request.URL.Path) {
			next.ServeHTTP(response, request)
			return
		}

		principal, err := verifyBearerToken(verifier, request.Header.Get("Authorization"))
		if err != nil {
			response.Header().Set(proxy.HeaderWWWAuthenticate, rejectedRequestChallenge)
			proxy.WriteError(response, request, proxy.ErrUnauthorized)

			return
		}

		scopes, known := proxy.RequiredScopes(request.Method, request.URL.Path)
		if !known {
			proxy.WriteError(response, request, proxy.ErrRouteNotFound)
			return
		}

		if !principal.HasScopes(scopes) {
			response.Header().Set(proxy.HeaderWWWAuthenticate,
				fmt.Sprintf(formatInsufficientScope, strings.Join(scopes, scopeSeparator)))
			proxy.WriteError(response, request, proxy.ErrInsufficientScope)

			return
		}

		request.Header.Set(HeaderUserID, principal.Subject)
		request.Header.Set(HeaderUserEmail, principal.Email)
		next.ServeHTTP(response, request)
	})
}

func verifyBearerToken(verifier TokenVerifier, header string) (proxy.Principal, error) {
	parts := strings.Fields(header)
	if len(parts) != authorizationHeaderParts || !strings.EqualFold(parts[0], bearerScheme) {
		return proxy.Principal{}, proxy.ErrUnauthorized
	}

	return verifier.Verify(parts[1])
}

func newRequestID() string {
	var value [requestIDBytes]byte

	_, _ = rand.Read(value[:])

	return hex.EncodeToString(value[:])
}
