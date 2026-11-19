package config

import (
	"testing"
	"time"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost/plot_db")
	t.Setenv("RABBITMQ_URL", "amqp://localhost")
	t.Setenv("OIDC_ISSUER", "http://localhost:8180/realms/land-plot")
	t.Setenv("OIDC_JWKS_URL", "http://keycloak:8080/realms/land-plot/protocol/openid-connect/certs")
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
		"OIDC_ISSUER":      " ",
		"OIDC_JWKS_URL":    "",
		"OIDC_TIMEOUT":     "0s",
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

func TestLoadReadsOIDCSettings(t *testing.T) {
	setRequiredEnv(t)

	serviceConfig, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if serviceConfig.OIDCIssuer != "http://localhost:8180/realms/land-plot" ||
		serviceConfig.OIDCAudience != defaultOIDCAudience || serviceConfig.OIDCTimeout != defaultOIDCTimeout {
		t.Fatalf("Load() OIDC settings = %q, %q, %v",
			serviceConfig.OIDCIssuer, serviceConfig.OIDCAudience, serviceConfig.OIDCTimeout)
	}
}

func TestLoadReadsAccuracySettings(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("ACCURACY_TREES_M", "4.5")
	t.Setenv("ACCURACY_FALLBACK_LAYER", "map")

	serviceConfig, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	accuracy := serviceConfig.Accuracy
	if accuracy.TreesM != 4.5 || accuracy.FallbackLayer != "map" ||
		accuracy.TouchClickPixels != defaultTouchClickPixels ||
		accuracy.FallbackZoom != defaultFallbackZoom ||
		accuracy.FallbackPointer != defaultFallbackPointer {
		t.Fatalf("Accuracy = %+v", accuracy)
	}

	for _, value := range []string{"0", "-1", "many", "NaN", "Inf"} {
		t.Setenv("ACCURACY_UNCLEAR_M", value)

		if _, err := Load(); err == nil {
			t.Errorf("ACCURACY_UNCLEAR_M=%q: Load() error = nil", value)
		}
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
