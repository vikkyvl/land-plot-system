package config

import "testing"

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost/identity_db")
	t.Setenv("OIDC_ISSUER", "http://localhost:8180/realms/land-plot")
	t.Setenv("OIDC_JWKS_URL", "http://keycloak:8080/realms/land-plot/protocol/openid-connect/certs")
	t.Setenv("SERVICE_INSTANCE_ID", "identity-1")
}

func TestLoadAppliesDefaults(t *testing.T) {
	setRequiredEnv(t)

	serviceConfig, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if serviceConfig.HTTPPort != 8081 || serviceConfig.OIDCAudience != defaultOIDCAudience ||
		serviceConfig.OIDCTimeout != defaultOIDCTimeout {
		t.Fatalf("Load() = %#v", serviceConfig)
	}
}

func TestLoadReadsOverrides(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("OIDC_AUDIENCE", "custom-api")

	serviceConfig, err := Load()
	if err != nil || serviceConfig.OIDCAudience != "custom-api" {
		t.Fatalf("Load() = %#v, %v", serviceConfig, err)
	}
}

func TestLoadRejectsInvalidSettings(t *testing.T) {
	tests := map[string]string{
		"DATABASE_URL":  "",
		"OIDC_ISSUER":   " ",
		"OIDC_JWKS_URL": "",
		"OIDC_TIMEOUT":  "0s",
		"HTTP_ADDRESS":  ":0",
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

func TestLoadReadsMetricsAddress(t *testing.T) {
	setRequiredEnv(t)

	serviceConfig, err := Load()
	if err != nil || serviceConfig.MetricsAddress != ":9464" {
		t.Fatalf("default MetricsAddress = %q, %v", serviceConfig.MetricsAddress, err)
	}

	t.Setenv("METRICS_ADDRESS", ":9500")

	if serviceConfig, err = Load(); err != nil || serviceConfig.MetricsAddress != ":9500" {
		t.Fatalf("MetricsAddress = %q, %v", serviceConfig.MetricsAddress, err)
	}

	t.Setenv("METRICS_ADDRESS", " ")

	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted an empty METRICS_ADDRESS")
	}
}
