package proxy

import (
	"net/http"
	"strings"
)

const (
	ScopePlotsRead    = "plots:read"
	ScopePlotsWrite   = "plots:write"
	ScopeReportsRead  = "reports:read"
	ScopeReportsWrite = "reports:write"
	ScopeOpenID       = "openid"
	pathSeparator     = "/"
	parameterPrefix   = "{"
	parameterSuffix   = "}"
)

type scopeRule struct {
	method  string
	pattern string
	scopes  []string
}

var scopeRules = []scopeRule{
	{method: http.MethodGet, pattern: "/api/v1/plots", scopes: []string{ScopePlotsRead}},
	{method: http.MethodGet, pattern: "/api/v1/plots/{id}", scopes: []string{ScopePlotsRead}},
	{method: http.MethodPost, pattern: "/api/v1/plots", scopes: []string{ScopePlotsWrite}},
	{method: http.MethodPatch, pattern: "/api/v1/plots/{id}", scopes: []string{ScopePlotsWrite}},
	{method: http.MethodGet, pattern: "/api/v1/reports/{id}", scopes: []string{ScopeReportsRead}},
	{method: http.MethodPost, pattern: "/api/v1/reports", scopes: []string{ScopeReportsWrite, ScopePlotsRead}},
	{method: http.MethodGet, pattern: "/api/v1/users/profile", scopes: []string{ScopeOpenID}},
	{method: http.MethodGet, pattern: "/api/v1/demo/chain"},
}

func RequiredScopes(method, path string) ([]string, bool) {
	for _, rule := range scopeRules {
		if rule.method == method && matchesPattern(rule.pattern, path) {
			return rule.scopes, true
		}
	}

	return nil, false
}

func (principal Principal) HasScopes(scopes []string) bool {
	for _, scope := range scopes {
		if !principal.HasScope(scope) {
			return false
		}
	}

	return true
}

func matchesPattern(pattern, path string) bool {
	patternSegments := strings.Split(pattern, pathSeparator)
	pathSegments := strings.Split(path, pathSeparator)

	if len(patternSegments) != len(pathSegments) {
		return false
	}

	for i, segment := range patternSegments {
		isParameter := strings.HasPrefix(segment, parameterPrefix) && strings.HasSuffix(segment, parameterSuffix)
		if isParameter && pathSegments[i] == "" || !isParameter && segment != pathSegments[i] {
			return false
		}
	}

	return true
}
