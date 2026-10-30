package config

import (
	"strings"
	"testing"
	"time"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost/identity_db")
	t.Setenv("JWT_SECRET", strings.Repeat("s", minimumJWTSecretLength))
	t.Setenv("SERVICE_INSTANCE_ID", "identity-1")
}

func TestLoadAppliesDefaults(t *testing.T) {
	setRequiredEnv(t)

	serviceConfig, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if serviceConfig.HTTPPort != 8081 || serviceConfig.TokenTTL != defaultTokenTTL ||
		serviceConfig.BcryptCost != defaultBcryptCost {
		t.Fatalf("Load() = %#v", serviceConfig)
	}
}

func TestLoadReadsOverrides(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("JWT_TTL", "15m")
	t.Setenv("BCRYPT_COST", "10")

	serviceConfig, err := Load()
	if err != nil || serviceConfig.TokenTTL != 15*time.Minute || serviceConfig.BcryptCost != 10 {
		t.Fatalf("Load() = %#v, %v", serviceConfig, err)
	}
}

func TestLoadRejectsInvalidSettings(t *testing.T) {
	tests := map[string]string{
		"DATABASE_URL": "",
		"JWT_SECRET":   strings.Repeat(" ", minimumJWTSecretLength),
		"HTTP_ADDRESS": ":0",
		"JWT_TTL":      "forever",
		"BCRYPT_COST":  "99",
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
