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

func TestLoadReadsSagaSettings(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("SAGA_TIMEOUT", "45s")
	t.Setenv("OUTBOX_POLL_INTERVAL", "500ms")

	serviceConfig, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if serviceConfig.SagaTimeout != 45*time.Second || serviceConfig.OutboxInterval != 500*time.Millisecond {
		t.Fatalf(
			"Load() saga timeout = %v, outbox interval = %v",
			serviceConfig.SagaTimeout,
			serviceConfig.OutboxInterval,
		)
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

func TestLoadReadsMessageProcessingTimeout(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("MESSAGE_PROCESSING_TIMEOUT", "7s")

	serviceConfig, err := Load()
	if err != nil || serviceConfig.ProcessingTimeout != 7*time.Second {
		t.Fatalf("Load() processing timeout = %v, %v", serviceConfig.ProcessingTimeout, err)
	}
}
