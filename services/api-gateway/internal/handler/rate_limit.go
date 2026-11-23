package handler

import (
	"math"
	"net"
	"net/http"
	"strconv"

	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/proxy"
)

const headerRetryAfter = "Retry-After"

func limitRequests(limiter RequestLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if !proxy.IsRateLimitedRoute(request.Method, request.URL.Path) {
			next.ServeHTTP(response, request)
			return
		}

		allowed, retryAfter := limiter.Allow(clientAddress(request))
		if !allowed {
			seconds := max(1, int(math.Ceil(retryAfter.Seconds())))
			response.Header().Set(headerRetryAfter, strconv.Itoa(seconds))
			proxy.WriteError(response, request, proxy.ErrRateLimited)

			return
		}

		next.ServeHTTP(response, request)
	})
}

func clientAddress(request *http.Request) string {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		return request.RemoteAddr
	}

	return host
}
