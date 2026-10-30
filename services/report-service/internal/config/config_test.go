package config

import (
	"strings"
	"testing"
	"time"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost/report_db")
	t.Setenv("RABBITMQ_URL", "amqp://localhost")
	t.Setenv("JWT_SECRET", strings.Repeat("s", minimumJWTSecretLength))
	t.Setenv("SERVICE_INSTANCE_ID", "report-1")
}

func TestLoadAppliesDefaults(t *testing.T) {
	setRequiredEnv(t)

	serviceConfig, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if serviceConfig.HTTPPort != 8083 || serviceConfig.EventPrefetch != defaultEventPrefetch ||
		serviceConfig.PlotServiceName != defaultPlotServiceName {
		t.Fatalf("Load() = %#v", serviceConfig)
	}
}

func TestLoadReadsOverrides(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("MESSAGE_PROCESSING_TIMEOUT", "7s")
	t.Setenv("RABBITMQ_PREFETCH", "3")

	serviceConfig, err := Load()
	if err != nil || serviceConfig.ProcessingTimeout != 7*time.Second || serviceConfig.EventPrefetch != 3 {
		t.Fatalf("Load() = %#v, %v", serviceConfig, err)
	}
}

func TestLoadRejectsInvalidSettings(t *testing.T) {
	tests := map[string]string{
		"RABBITMQ_URL":           " ",
		"LAND_PLOT_SERVICE_NAME": "",
		"JWT_SECRET":             "short",
		"HTTP_ADDRESS":           "no-port",
		"UPSTREAM_TIMEOUT":       "0s",
		"RABBITMQ_PREFETCH":      "0",
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
