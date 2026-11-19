package handler

import (
	"time"

	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/proxy"
)

type TokenVerifier interface {
	Verify(token string) (proxy.Principal, error)
}

type RequestLimiter interface {
	Allow(key string) (bool, time.Duration)
}
