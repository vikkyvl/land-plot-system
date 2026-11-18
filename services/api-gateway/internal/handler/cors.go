package handler

import (
	"net/http"
	"slices"

	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/proxy"
)

const (
	headerOrigin                = "Origin"
	headerVary                  = "Vary"
	headerAllowOrigin           = "Access-Control-Allow-Origin"
	headerAllowMethods          = "Access-Control-Allow-Methods"
	headerAllowHeaders          = "Access-Control-Allow-Headers"
	headerExposeHeaders         = "Access-Control-Expose-Headers"
	headerMaxAge                = "Access-Control-Max-Age"
	headerRequestMethod         = "Access-Control-Request-Method"
	headerRequestPrivateNetwork = "Access-Control-Request-Private-Network"
	headerAllowPrivateNetwork   = "Access-Control-Allow-Private-Network"
	allowedMethods              = "GET, POST, PATCH, DELETE, OPTIONS"
	allowedHeaders              = "Authorization, Content-Type, X-Request-ID"
	exposedHeaders              = "X-Request-ID"
	preflightMaxAgeSeconds      = "600"
	privateNetworkAllowed       = "true"
	privateNetworkRequested     = "true"
)

func withCORS(allowedOrigins []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		origin := request.Header.Get(headerOrigin)
		if origin == "" {
			next.ServeHTTP(response, request)
			return
		}

		response.Header().Add(headerVary, headerOrigin)
		preflight := request.Method == http.MethodOptions && request.Header.Get(headerRequestMethod) != ""

		if !slices.Contains(allowedOrigins, origin) {
			if preflight {
				proxy.WriteError(response, request, proxy.ErrOriginNotAllowed)
				return
			}

			next.ServeHTTP(response, request)

			return
		}

		response.Header().Set(headerAllowOrigin, origin)
		response.Header().Set(headerExposeHeaders, exposedHeaders)

		if !preflight {
			next.ServeHTTP(response, request)
			return
		}

		response.Header().Set(headerAllowMethods, allowedMethods)
		response.Header().Set(headerAllowHeaders, allowedHeaders)
		response.Header().Set(headerMaxAge, preflightMaxAgeSeconds)

		if request.Header.Get(headerRequestPrivateNetwork) == privateNetworkRequested {
			response.Header().Set(headerAllowPrivateNetwork, privateNetworkAllowed)
		}

		response.WriteHeader(http.StatusNoContent)
	})
}
