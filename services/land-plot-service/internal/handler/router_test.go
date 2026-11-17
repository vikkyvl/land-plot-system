package handler

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/handler/respond"
)

func TestPlotRoutesRequireBearerToken(t *testing.T) {
	for _, target := range []string{"/api/v1/plots", "/api/v1/plots/plot-1", "/internal/plots/plot-1"} {
		response := serve(newTestRouter(&plotServiceStub{}), httptest.NewRequest(http.MethodGet, target, http.NoBody))

		if response.Code != http.StatusUnauthorized {
			t.Fatalf("GET %s status = %d, want %d", target, response.Code, http.StatusUnauthorized)
		}
	}
}

func TestOwnerComesFromVerifiedTokenNotHeader(t *testing.T) {
	plots := &plotServiceStub{}
	request := authorizedRequest(http.MethodPost, "/api/v1/plots", testPlotBody)
	request.Header.Set("X-User-ID", "forged-owner")

	serve(newTestRouter(plots), request)

	if plots.ownerID != testOwnerID {
		t.Fatalf("owner ID = %q, want %q", plots.ownerID, testOwnerID)
	}
}

func TestServiceErrorsReachTheClient(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{domain.ErrSelfIntersection, http.StatusUnprocessableEntity, "self_intersection"},
		{domain.ErrVersionConflict, http.StatusConflict, "version_conflict"},
	}
	for _, test := range tests {
		t.Run(test.wantCode, func(t *testing.T) {
			request := authorizedRequest(http.MethodPost, "/api/v1/plots", testPlotBody)
			request.Header.Set(respond.RequestIDHeader, "request-1")

			response := serve(newTestRouter(&plotServiceStub{err: test.err}), request)

			body := decodeBody[respond.ErrorResponse](t, response)
			if response.Code != test.wantStatus || body.Code != test.wantCode || body.RequestID != "request-1" {
				t.Fatalf("status = %d body = %#v", response.Code, body)
			}
		})
	}
}

func TestEveryRouteRequiresItsScope(t *testing.T) {
	tests := []struct {
		method string
		target string
		body   string
		scope  string
	}{
		{http.MethodPost, "/api/v1/plots", testPlotBody, domain.ScopePlotsWrite},
		{http.MethodGet, "/api/v1/plots", "", domain.ScopePlotsRead},
		{http.MethodGet, "/api/v1/plots/plot-1", "", domain.ScopePlotsRead},
		{http.MethodPatch, "/api/v1/plots/plot-1", `{"name":"Нова"}`, domain.ScopePlotsWrite},
		{http.MethodGet, "/internal/plots/plot-1", "", domain.ScopePlotsRead},
	}
	for _, test := range tests {
		t.Run(test.method+" "+test.target, func(t *testing.T) {
			otherScopes := slices.DeleteFunc(
				slices.Clone(allScopes),
				func(scope string) bool { return scope == test.scope },
			)

			denied := serve(newTestRouterWithScopes(&plotServiceStub{}, otherScopes...),
				authorizedRequest(test.method, test.target, test.body))

			body := decodeBody[respond.ErrorResponse](t, denied)
			if denied.Code != http.StatusForbidden || body.Code != "insufficient_scope" ||
				denied.Header().Get("WWW-Authenticate") != `Bearer error="insufficient_scope", scope="`+test.scope+`"` {
				t.Fatalf("without %s: status = %d, body = %#v, challenge = %q",
					test.scope, denied.Code, body, denied.Header().Get("WWW-Authenticate"))
			}

			allowed := serve(newTestRouterWithScopes(&plotServiceStub{}, test.scope),
				authorizedRequest(test.method, test.target, test.body))
			if allowed.Code == http.StatusForbidden || allowed.Code == http.StatusUnauthorized {
				t.Fatalf("with %s: status = %d", test.scope, allowed.Code)
			}
		})
	}
}

func TestMissingTokenGetsInvalidTokenChallenge(t *testing.T) {
	response := serve(
		newTestRouter(&plotServiceStub{}),
		httptest.NewRequest(http.MethodGet, "/api/v1/plots", http.NoBody),
	)

	if response.Code != http.StatusUnauthorized ||
		response.Header().Get("WWW-Authenticate") != `Bearer error="invalid_token"` {
		t.Fatalf("status = %d, challenge = %q", response.Code, response.Header().Get("WWW-Authenticate"))
	}
}

func TestDemoChainNeedsATokenButNoScope(t *testing.T) {
	anonymous := serve(
		newTestRouter(&plotServiceStub{}),
		httptest.NewRequest(http.MethodGet, "/internal/demo/chain", http.NoBody),
	)
	if anonymous.Code != http.StatusUnauthorized {
		t.Fatalf("without token status = %d", anonymous.Code)
	}

	authenticated := serve(newTestRouterWithScopes(&plotServiceStub{}),
		authorizedRequest(http.MethodGet, "/internal/demo/chain", ""))
	if authenticated.Code != http.StatusOK {
		t.Fatalf("with a token without scopes status = %d", authenticated.Code)
	}
}

func TestHealthStaysPublic(t *testing.T) {
	if response := serve(
		newTestRouter(&plotServiceStub{}),
		httptest.NewRequest(http.MethodGet, "/health", http.NoBody),
	); response.Code != http.StatusOK {
		t.Fatalf("GET /health status = %d", response.Code)
	}
}
