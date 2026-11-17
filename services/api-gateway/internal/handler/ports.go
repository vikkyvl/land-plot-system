package handler

import "github.com/vleshchenko/land-plot-system/services/api-gateway/internal/proxy"

type TokenVerifier interface {
	Verify(token string) (proxy.Principal, error)
}
