package handler

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/handler/respond"
)

func routerWithScopes(reports ReportService, scopes ...string) http.Handler {
	return newTestRouterWithVerifier(reports, verifierStub{userID: testOwnerID, scopes: scopes})
}

func TestEveryRouteRequiresItsScopes(t *testing.T) {
	createBody := `{"plot_id":"7d1f0a2e-3b4c-4d5e-8f60-718293a4b5c6"}`

	tests := []struct {
		method string
		target string
		body   string
		scopes []string
	}{
		{http.MethodPost, "/api/v1/reports", createBody, []string{domain.ScopeReportsWrite, domain.ScopePlotsRead}},
		{http.MethodGet, "/api/v1/reports/report-1", "", []string{domain.ScopeReportsRead}},
		{http.MethodGet, "/api/v1/reports", "", []string{domain.ScopeReportsRead}},
	}
	for _, test := range tests {
		for _, missing := range test.scopes {
			t.Run(test.method+" "+test.target+" without "+missing, func(t *testing.T) {
				granted := slices.DeleteFunc(
					slices.Clone(allScopes),
					func(scope string) bool { return scope == missing },
				)

				response := serve(routerWithScopes(&reportServiceStub{}, granted...),
					authorizedRequest(test.method, test.target, test.body))

				body := decodeBody[respond.ErrorResponse](t, response)

				wantChallenge := `Bearer error="insufficient_scope", scope="` + strings.Join(test.scopes, " ") + `"`
				if response.Code != http.StatusForbidden || body.Code != "insufficient_scope" ||
					response.Header().Get("WWW-Authenticate") != wantChallenge {
					t.Fatalf("status = %d, body = %#v, challenge = %q",
						response.Code, body, response.Header().Get("WWW-Authenticate"))
				}
			})
		}

		t.Run(test.method+" "+test.target+" with its scopes", func(t *testing.T) {
			response := serve(routerWithScopes(&reportServiceStub{}, test.scopes...),
				authorizedRequest(test.method, test.target, test.body))
			if response.Code == http.StatusForbidden || response.Code == http.StatusUnauthorized {
				t.Fatalf("status = %d", response.Code)
			}
		})
	}
}

func TestLandPlotRefusalReachesTheUserAsForbidden(t *testing.T) {
	reports := &reportServiceStub{err: domain.ErrInsufficientScope}

	response := serve(newTestRouter(reports),
		authorizedRequest(http.MethodPost, "/api/v1/reports", `{"plot_id":"7d1f0a2e-3b4c-4d5e-8f60-718293a4b5c6"}`))

	body := decodeBody[respond.ErrorResponse](t, response)
	if response.Code != http.StatusForbidden || body.Code != "insufficient_scope" {
		t.Fatalf("status = %d, body = %#v", response.Code, body)
	}
}

func TestMissingTokenGetsInvalidTokenChallenge(t *testing.T) {
	response := serve(newTestRouter(&reportServiceStub{}), withoutToken(http.MethodGet, "/api/v1/reports/report-1"))

	if response.Code != http.StatusUnauthorized ||
		response.Header().Get("WWW-Authenticate") != `Bearer error="invalid_token"` {
		t.Fatalf("status = %d, challenge = %q", response.Code, response.Header().Get("WWW-Authenticate"))
	}
}

func withoutToken(method, target string) *http.Request {
	request := authorizedRequest(method, target, "")
	request.Header.Del("Authorization")

	return request
}
