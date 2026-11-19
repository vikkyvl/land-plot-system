package config

import (
	"slices"
	"testing"
	"time"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("OIDC_ISSUER", "http://localhost:8180/realms/land-plot")
	t.Setenv("OIDC_JWKS_URL", "http://keycloak:8080/realms/land-plot/protocol/openid-connect/certs")
	t.Setenv("SERVICE_INSTANCE_ID", "gateway-1")
}

func TestLoadAppliesDefaults(t *testing.T) {
	setRequiredEnv(t)

	serviceConfig, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if serviceConfig.HTTPPort != 8080 || serviceConfig.UpstreamTimeout != defaultUpstreamTimeout ||
		serviceConfig.InstanceID != "gateway-1" || serviceConfig.PreviewRateLimitPerMinute != 60 {
		t.Fatalf("Load() = %#v", serviceConfig)
	}
}

func TestLoadReadsUpstreamTimeout(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("UPSTREAM_TIMEOUT", "2s")

	serviceConfig, err := Load()
	if err != nil || serviceConfig.UpstreamTimeout != 2*time.Second {
		t.Fatalf("Load() = %v, %v", serviceConfig.UpstreamTimeout, err)
	}
}

func TestLoadReadsPreviewRateLimit(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("PREVIEW_RATE_LIMIT_PER_MINUTE", "5")

	serviceConfig, err := Load()
	if err != nil || serviceConfig.PreviewRateLimitPerMinute != 5 {
		t.Fatalf("Load() = %v, %v", serviceConfig.PreviewRateLimitPerMinute, err)
	}
}

func TestLoadRejectsInvalidSettings(t *testing.T) {
	tests := map[string]string{
		"CONSUL_ADDRESS":   " ",
		"OIDC_ISSUER":      " ",
		"OIDC_JWKS_URL":    "",
		"OIDC_TIMEOUT":     "0s",
		"HTTP_ADDRESS":     ":http-port",
		"UPSTREAM_TIMEOUT": "-5s",

		"PREVIEW_RATE_LIMIT_PER_MINUTE": "0",
	}
	for key, value := range tests {
		t.Run(key, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv(key, value)

			if _, err := Load(); err == nil {
				t.Fatalf("Load() with %s=%q succeeded", key, value)
			}
		})
	}
}

func TestLoadReadsAllowedOrigins(t *testing.T) {
	setRequiredEnv(t)

	serviceConfig, err := Load()
	if err != nil || !slices.Equal(serviceConfig.AllowedOrigins,
		[]string{"https://cad.kpi.ua", "http://localhost:5173", "http://localhost:4173"}) {
		t.Fatalf("Load() origins = %v, %v", serviceConfig.AllowedOrigins, err)
	}

	t.Setenv("CORS_ALLOWED_ORIGINS", " https://cad.kpi.ua , ,http://localhost:5173")

	serviceConfig, err = Load()
	if err != nil ||
		!slices.Equal(serviceConfig.AllowedOrigins, []string{"https://cad.kpi.ua", "http://localhost:5173"}) {
		t.Fatalf("Load() origins = %v, %v", serviceConfig.AllowedOrigins, err)
	}
}

func TestLoadRejectsInvalidOrigins(t *testing.T) {
	for _, value := range []string{"*", "https://cad.kpi.ua/", "https://cad.kpi.ua/api", "cad.kpi.ua", "ftp://cad.kpi.ua"} {
		t.Run(value, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv("CORS_ALLOWED_ORIGINS", value)

			if _, err := Load(); err == nil {
				t.Fatalf("Load() with CORS_ALLOWED_ORIGINS=%q succeeded", value)
			}
		})
	}
}
