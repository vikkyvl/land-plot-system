package config

import (
	"testing"
	"time"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost/report_db")
	t.Setenv("RABBITMQ_URL", "amqp://localhost")
	t.Setenv("OIDC_ISSUER", "http://localhost:8180/realms/land-plot")
	t.Setenv("OIDC_JWKS_URL", "http://keycloak:8080/realms/land-plot/protocol/openid-connect/certs")
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
		"OIDC_ISSUER":            " ",
		"OIDC_JWKS_URL":          "",
		"OIDC_TIMEOUT":           "0s",
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

func TestLoadReadsExternalDataSettings(t *testing.T) {
	setRequiredEnv(t)

	serviceConfig, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if serviceConfig.OpenMeteo.ElevationURL != defaultElevationURL ||
		serviceConfig.OpenMeteo.ArchiveURL != defaultArchiveURL ||
		serviceConfig.Climate != (ClimateConfig{Years: 10, CacheTTL: 8760 * time.Hour}) ||
		serviceConfig.OpenMeteo.Timeout != defaultOpenMeteoTimeout ||
		serviceConfig.Terrain != (TerrainConfig{GridSide: 10, MarginM: 90, CoarseAreaM2: 10_000}) {
		t.Fatalf("Load() = %+v, %+v", serviceConfig.OpenMeteo, serviceConfig.Terrain)
	}

	invalid := map[string]string{
		"OPEN_METEO_ELEVATION_URL": "ftp://example.org",
		"OPEN_METEO_TIMEOUT":       "0s",
		"TERRAIN_GRID_SIDE":        "11",
		"TERRAIN_MARGIN_M":         "-5",
		"TERRAIN_COARSE_AREA_M2":   "lots",
		"OPEN_METEO_ARCHIVE_URL":   "archive",
		"CLIMATE_YEARS":            "0",
		"CLIMATE_CACHE_TTL":        "a year",
	}
	for key, value := range invalid {
		t.Run(key, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv(key, value)

			if _, err := Load(); err == nil {
				t.Fatalf("%s=%q: Load() error = nil", key, value)
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
