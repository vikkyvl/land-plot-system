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

func TestLoadReadsOutboxAndFailureSettings(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("OUTBOX_POLL_INTERVAL", "500ms")
	t.Setenv("REPORT_FAIL_NAME_PREFIX", "FAIL")

	serviceConfig, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if serviceConfig.OutboxInterval != 500*time.Millisecond || serviceConfig.FailNamePrefix != "FAIL" {
		t.Fatalf("Load() = %#v", serviceConfig)
	}
}

func TestLoadReadsOutboxMaxAttempts(t *testing.T) {
	setRequiredEnv(t)

	serviceConfig, err := Load()
	if err != nil || serviceConfig.OutboxAttempts != defaultOutboxAttempts {
		t.Fatalf("Load() default attempts = %d, %v", serviceConfig.OutboxAttempts, err)
	}

	t.Setenv("OUTBOX_MAX_ATTEMPTS", "3")

	if serviceConfig, err = Load(); err != nil || serviceConfig.OutboxAttempts != 3 {
		t.Fatalf("Load() attempts = %d, %v", serviceConfig.OutboxAttempts, err)
	}

	t.Setenv("OUTBOX_MAX_ATTEMPTS", "0")

	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted zero OUTBOX_MAX_ATTEMPTS")
	}
}

func TestLoadReadsOutboxCleanupSettings(t *testing.T) {
	setRequiredEnv(t)

	serviceConfig, err := Load()
	if err != nil || serviceConfig.OutboxCleanupInterval != time.Minute || serviceConfig.OutboxRetention != time.Hour {
		t.Fatalf(
			"Load() defaults = %v, %v, %v",
			serviceConfig.OutboxCleanupInterval,
			serviceConfig.OutboxRetention,
			err,
		)
	}

	t.Setenv("OUTBOX_CLEANUP_INTERVAL", "30s")
	t.Setenv("OUTBOX_RETENTION", "2h")

	serviceConfig, err = Load()
	if err != nil || serviceConfig.OutboxCleanupInterval != 30*time.Second ||
		serviceConfig.OutboxRetention != 2*time.Hour {
		t.Fatalf("Load() = %v, %v, %v", serviceConfig.OutboxCleanupInterval, serviceConfig.OutboxRetention, err)
	}
}
