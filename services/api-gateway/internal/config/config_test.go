package config

import (
	"strings"
	"testing"
	"time"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("JWT_SECRET", strings.Repeat("s", minimumJWTSecretLength))
	t.Setenv("SERVICE_INSTANCE_ID", "gateway-1")
}

func TestLoadAppliesDefaults(t *testing.T) {
	setRequiredEnv(t)

	serviceConfig, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if serviceConfig.HTTPPort != 8080 || serviceConfig.UpstreamTimeout != defaultUpstreamTimeout ||
		serviceConfig.InstanceID != "gateway-1" {
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

func TestLoadRejectsInvalidSettings(t *testing.T) {
	tests := map[string]string{
		"CONSUL_ADDRESS":   " ",
		"JWT_SECRET":       "short",
		"HTTP_ADDRESS":     ":http-port",
		"UPSTREAM_TIMEOUT": "-5s",
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
