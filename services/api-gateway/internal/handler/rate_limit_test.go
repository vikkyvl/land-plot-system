package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/proxy"
)

const previewPath = "/api/v1/plots/preview"

func previewRequest() *http.Request {
	request := httptest.NewRequest(http.MethodPost, previewPath, strings.NewReader(`{"points":[]}`))
	request.RemoteAddr = "192.0.2.10:51234"

	return request
}

func TestPreviewReachesUpstreamWithoutToken(t *testing.T) {
	upstreamCalled := false
	upstream := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		upstreamCalled = true

		if request.Header.Get(HeaderUserID) != "" {
			t.Errorf("%s = %q, want empty", HeaderUserID, request.Header.Get(HeaderUserID))
		}

		response.WriteHeader(http.StatusOK)
	})

	response := serve(newTestRouter(upstream, verifierStub{err: errors.New("no token")}), previewRequest())

	if response.Code != http.StatusOK || !upstreamCalled {
		t.Fatalf("status = %d, upstream called = %v", response.Code, upstreamCalled)
	}
}

func TestPreviewIgnoresAnInvalidToken(t *testing.T) {
	upstream := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusOK)
	})
	request := previewRequest()
	request.Header.Set("Authorization", "Bearer forged-token")
	request.Header.Set(HeaderUserID, "attacker")

	response := serve(newTestRouter(upstream, verifierStub{err: errors.New("invalid")}), request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestPreviewOverTheLimitGetsTooManyRequests(t *testing.T) {
	upstream := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("upstream called") })
	limiter := &limiterStub{retryAfter: 1500 * time.Millisecond}

	response := serve(newTestRouterWithLimiter(upstream, verifierStub{}, limiter), previewRequest())

	body := decodeError(t, response)
	if response.Code != http.StatusTooManyRequests || body.Code != proxy.ErrRateLimited.Code ||
		response.Header().Get(headerRetryAfter) != "2" || body.RequestID == "" {
		t.Fatalf("status = %d, Retry-After = %q, body = %#v",
			response.Code, response.Header().Get(headerRetryAfter), body)
	}

	if len(limiter.keys) != 1 || limiter.keys[0] != "192.0.2.10" {
		t.Fatalf("limiter keys = %v, want the client IP without port", limiter.keys)
	}
}

func TestBrowserCanReadRetryAfterOfALimitedPreview(t *testing.T) {
	upstream := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("upstream called") })
	request := previewRequest()
	request.Header.Set("Origin", allowedOrigin)

	response := serve(
		newTestRouterWithLimiter(upstream, verifierStub{}, &limiterStub{retryAfter: 3 * time.Second}),
		request,
	)

	exposed := strings.Split(response.Header().Get("Access-Control-Expose-Headers"), ", ")
	if response.Code != http.StatusTooManyRequests ||
		response.Header().Get("Access-Control-Allow-Origin") != allowedOrigin ||
		response.Header().Get(headerRetryAfter) != "3" || !slices.Contains(exposed, headerRetryAfter) {
		t.Fatalf("status = %d, headers = %#v", response.Code, response.Header())
	}
}

func TestLimitAppliesOnlyToPreview(t *testing.T) {
	upstream := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusOK)
	})
	limiter := &limiterStub{}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/plots", http.NoBody)
	request.Header.Set("Authorization", "Bearer valid-token")

	response := serve(newTestRouterWithLimiter(upstream, verifierStub{scopes: []string{proxy.ScopePlotsRead}}, limiter),
		request)

	if response.Code != http.StatusOK || len(limiter.keys) != 0 {
		t.Fatalf("status = %d, limiter keys = %v", response.Code, limiter.keys)
	}
}

func TestPreviewPreflightIsNotLimited(t *testing.T) {
	limiter := &limiterStub{}
	request := httptest.NewRequest(http.MethodOptions, previewPath, http.NoBody)
	request.Header.Set("Origin", allowedOrigin)
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)

	response := serve(newTestRouterWithLimiter(http.NotFoundHandler(), verifierStub{}, limiter), request)

	if response.Code != http.StatusNoContent || len(limiter.keys) != 0 {
		t.Fatalf("status = %d, limiter keys = %v", response.Code, limiter.keys)
	}
}
