package domain

import "slices"

type Principal struct {
	Subject string
	Email   string
	Name    string
	Scopes  []string
}

func (principal Principal) HasScope(scope string) bool {
	return slices.Contains(principal.Scopes, scope)
}
