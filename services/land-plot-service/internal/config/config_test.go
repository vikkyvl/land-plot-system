package config

import (
	"strings"
	"testing"
	"time"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost/plot_db")
	t.Setenv("RABBITMQ_URL", "amqp://localhost")
	t.Setenv("JWT_SECRET", strings.Repeat("s", minimumJWTSecretLength))
	t.Setenv("SERVICE_INSTANCE_ID", "plot-1")
}

func TestLoadAppliesDefaults(t *testing.T) {
	setRequiredEnv(t)

	serviceConfig, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if serviceConfig.HTTPPort != 8082 || serviceConfig.ShutdownTimeout != defaultShutdownTimeout ||
		serviceConfig.InstanceID != "plot-1" {
		t.Fatalf("Load() = %#v", serviceConfig)
	}
}

func TestLoadReadsDurations(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("RABBITMQ_TIMEOUT", "7s")

	serviceConfig, err := Load()
	if err != nil || serviceConfig.RabbitMQTimeout != 7*time.Second {
		t.Fatalf("Load() = %v, %v", serviceConfig.RabbitMQTimeout, err)
	}
}

func TestLoadRejectsInvalidSettings(t *testing.T) {
	tests := map[string]string{
		"DATABASE_URL":     " ",
		"JWT_SECRET":       "short",
		"HTTP_ADDRESS":     ":99999",
		"CONSUL_TIMEOUT":   "-1s",
		"DATABASE_TIMEOUT": "soon",
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
