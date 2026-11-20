package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/auth"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/config"
)

func newTokenVerifier(
	serviceConfig config.Config,
	logger *slog.Logger,
) (*auth.OIDCVerifier, context.CancelFunc, error) {
	verifierContext, stopVerifier := context.WithCancel(context.Background())

	verifier, err := auth.NewOIDCVerifier(verifierContext, auth.OIDCSettings{
		JWKSURL:  serviceConfig.OIDCJWKSURL,
		Issuer:   serviceConfig.OIDCIssuer,
		Audience: serviceConfig.OIDCAudience,
		Timeout:  serviceConfig.OIDCTimeout,
	}, logger)
	if err != nil {
		stopVerifier()
		return nil, nil, fmt.Errorf(wrapCreateTokenVerifier, err)
	}

	return verifier, stopVerifier, nil
}
