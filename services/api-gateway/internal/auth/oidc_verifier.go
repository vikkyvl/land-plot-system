package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/MicahParks/jwkset"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/time/rate"

	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/logging"
	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/proxy"
)

const (
	signingAlgorithm        = "RS256"
	clockLeeway             = 30 * time.Second
	keyRefreshInterval      = time.Hour
	unknownKeyRefreshPeriod = 30 * time.Second
	unknownKeyRefreshWait   = time.Second
)

type OIDCSettings struct {
	JWKSURL  string
	Issuer   string
	Audience string
	Timeout  time.Duration
}

type tokenClaims struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Scope string `json:"scope"`
	jwt.RegisteredClaims
}

type OIDCVerifier struct {
	keys     jwt.Keyfunc
	settings OIDCSettings
	now      func() time.Time
}

func NewOIDCVerifier(
	verifierContext context.Context,
	settings OIDCSettings,
	logger *slog.Logger,
) (*OIDCVerifier, error) {
	remoteKeys, err := jwkset.NewStorageFromHTTP(settings.JWKSURL, jwkset.HTTPClientStorageOptions{
		Client:                    &http.Client{Timeout: settings.Timeout},
		Ctx:                       verifierContext,
		HTTPTimeout:               settings.Timeout,
		NoErrorReturnFirstHTTPReq: true,
		RefreshInterval:           keyRefreshInterval,
		RefreshErrorHandler: func(_ context.Context, err error) {
			logger.Warn(logRefreshSigningKeys, logging.KeyError, err)
		},
	})
	if err != nil {
		return nil, fmt.Errorf(wrapCreateKeyStorage, err)
	}

	storage, err := jwkset.NewHTTPClient(jwkset.HTTPClientOptions{
		HTTPURLs:          map[string]jwkset.Storage{settings.JWKSURL: remoteKeys},
		RateLimitWaitMax:  unknownKeyRefreshWait,
		RefreshUnknownKID: rate.NewLimiter(rate.Every(unknownKeyRefreshPeriod), 1),
	})
	if err != nil {
		return nil, fmt.Errorf(wrapCreateKeyStorage, err)
	}

	keys, err := keyfunc.New(keyfunc.Options{
		Ctx:          verifierContext,
		Storage:      storage,
		UseWhitelist: []jwkset.USE{jwkset.UseSig},
	})
	if err != nil {
		return nil, fmt.Errorf(wrapCreateKeyfunc, err)
	}

	return &OIDCVerifier{keys: keys.Keyfunc, settings: settings, now: time.Now}, nil
}

func (verifier *OIDCVerifier) Verify(token string) (proxy.Principal, error) {
	var claims tokenClaims

	parsed, err := jwt.ParseWithClaims(token, &claims, verifier.keys,
		jwt.WithValidMethods([]string{signingAlgorithm}),
		jwt.WithIssuer(verifier.settings.Issuer),
		jwt.WithAudience(verifier.settings.Audience),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(clockLeeway),
		jwt.WithTimeFunc(verifier.now),
	)
	if errors.Is(err, jwt.ErrTokenExpired) {
		return proxy.Principal{}, fmt.Errorf(formatWrapTokenError, ErrExpiredToken, err)
	}

	if err != nil {
		return proxy.Principal{}, fmt.Errorf(formatWrapTokenError, ErrInvalidToken, err)
	}

	if !parsed.Valid || claims.Subject == "" {
		return proxy.Principal{}, ErrInvalidToken
	}

	return proxy.Principal{
		Subject: claims.Subject,
		Email:   claims.Email,
		Name:    claims.Name,
		Scopes:  strings.Fields(claims.Scope),
	}, nil
}
