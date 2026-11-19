package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const ServiceName = "land-plot-service"

const (
	defaultHTTPAddress     = ":8082"
	defaultMetricsAddress  = ":9464"
	defaultConsulAddress   = "http://localhost:8500"
	defaultShutdownTimeout = 10 * time.Second
	defaultConsulTimeout   = 3 * time.Second
	defaultDatabaseTimeout = 5 * time.Second
	defaultRabbitMQTimeout = 5 * time.Second
	defaultOutboxInterval  = time.Second
	defaultSagaTimeout     = 30 * time.Second
	defaultOutboxAttempts  = 10
	defaultOutboxCleanup   = time.Minute
	defaultOutboxRetention = time.Hour
	defaultProcessingTime  = 5 * time.Second
	defaultOIDCAudience    = "land-plot-api"
	defaultOIDCTimeout     = 5 * time.Second
	maximumPort            = 65535
)

const (
	defaultMouseClickPixels = 3
	defaultTouchClickPixels = 8
	defaultMapShiftM        = 2
	defaultSatelliteShiftM  = 4
	defaultTreesM           = 3
	defaultBuildingsM       = 2
	defaultUnclearM         = 5
	defaultFallbackZoom     = 16
	defaultFallbackLayer    = "satellite"
	defaultFallbackPointer  = "touch"
)

var errInvalidPort = errors.New(envHTTPAddress + " must contain a valid port")

type Config struct {
	HTTPAddress     string
	MetricsAddress  string
	OTLPEndpoint    string
	HTTPPort        int
	ShutdownTimeout time.Duration
	ConsulAddress   string
	ConsulTimeout   time.Duration
	ServiceAddress  string
	InstanceID      string
	DatabaseURL     string
	DatabaseTimeout time.Duration
	RabbitMQURL     string
	RabbitMQTimeout time.Duration
	OutboxInterval  time.Duration
	SagaTimeout     time.Duration
	OutboxAttempts  int

	OutboxCleanupInterval time.Duration
	OutboxRetention       time.Duration
	ProcessingTimeout     time.Duration

	OIDCIssuer   string
	OIDCJWKSURL  string
	OIDCAudience string
	OIDCTimeout  time.Duration

	Accuracy AccuracyConfig
}

type AccuracyConfig struct {
	MouseClickPixels float64
	TouchClickPixels float64
	MapShiftM        float64
	SatelliteShiftM  float64
	TreesM           float64
	BuildingsM       float64
	UnclearM         float64
	FallbackZoom     float64
	FallbackLayer    string
	FallbackPointer  string
}

func Load() (Config, error) {
	serviceConfig := Config{
		HTTPAddress:    stringOrDefault(envHTTPAddress, defaultHTTPAddress),
		MetricsAddress: stringOrDefault(envMetricsAddress, defaultMetricsAddress),
		OTLPEndpoint:   os.Getenv(envOTLPEndpoint),
		ConsulAddress:  stringOrDefault(envConsulAddress, defaultConsulAddress),
		ServiceAddress: os.Getenv(envServiceAddress),
		InstanceID:     os.Getenv(envServiceInstanceID),
		DatabaseURL:    os.Getenv(envDatabaseURL),
		OIDCIssuer:     os.Getenv(envOIDCIssuer),
		OIDCJWKSURL:    os.Getenv(envOIDCJWKSURL),
		OIDCAudience:   stringOrDefault(envOIDCAudience, defaultOIDCAudience),
		RabbitMQURL:    os.Getenv(envRabbitMQURL),
	}
	if err := serviceConfig.loadDurations(); err != nil {
		return Config{}, err
	}

	if err := serviceConfig.validate(); err != nil {
		return Config{}, err
	}

	if err := serviceConfig.loadAccuracy(); err != nil {
		return Config{}, err
	}

	var err error
	if serviceConfig.OutboxAttempts, err = positiveIntOrDefault(
		envOutboxMaxAttempts,
		defaultOutboxAttempts,
	); err != nil {
		return Config{}, err
	}

	if serviceConfig.HTTPPort, err = parsePort(serviceConfig.HTTPAddress); err != nil {
		return Config{}, err
	}

	if serviceConfig.InstanceID == "" {
		if serviceConfig.InstanceID, err = defaultInstanceID(); err != nil {
			return Config{}, err
		}
	}

	return serviceConfig, nil
}

func (serviceConfig *Config) loadDurations() error {
	durations := []struct {
		key      string
		target   *time.Duration
		fallback time.Duration
	}{
		{envHTTPShutdownTimeout, &serviceConfig.ShutdownTimeout, defaultShutdownTimeout},
		{envConsulTimeout, &serviceConfig.ConsulTimeout, defaultConsulTimeout},
		{envDatabaseTimeout, &serviceConfig.DatabaseTimeout, defaultDatabaseTimeout},
		{envRabbitMQTimeout, &serviceConfig.RabbitMQTimeout, defaultRabbitMQTimeout},
		{envOutboxPollInterval, &serviceConfig.OutboxInterval, defaultOutboxInterval},
		{envSagaTimeout, &serviceConfig.SagaTimeout, defaultSagaTimeout},
		{envOutboxCleanupInterval, &serviceConfig.OutboxCleanupInterval, defaultOutboxCleanup},
		{envOutboxRetention, &serviceConfig.OutboxRetention, defaultOutboxRetention},
		{envMessageProcessingTimeout, &serviceConfig.ProcessingTimeout, defaultProcessingTime},
		{envOIDCTimeout, &serviceConfig.OIDCTimeout, defaultOIDCTimeout},
	}
	for _, duration := range durations {
		value, err := durationOrDefault(duration.key, duration.fallback)
		if err != nil {
			return err
		}

		*duration.target = value
	}

	return nil
}

func (serviceConfig *Config) loadAccuracy() error {
	accuracy := &serviceConfig.Accuracy
	accuracy.FallbackLayer = stringOrDefault(envAccuracyFallbackLayer, defaultFallbackLayer)
	accuracy.FallbackPointer = stringOrDefault(envAccuracyFallbackPointer, defaultFallbackPointer)

	numbers := []struct {
		key      string
		target   *float64
		fallback float64
	}{
		{envAccuracyMouseClickPixels, &accuracy.MouseClickPixels, defaultMouseClickPixels},
		{envAccuracyTouchClickPixels, &accuracy.TouchClickPixels, defaultTouchClickPixels},
		{envAccuracyMapShiftM, &accuracy.MapShiftM, defaultMapShiftM},
		{envAccuracySatelliteShiftM, &accuracy.SatelliteShiftM, defaultSatelliteShiftM},
		{envAccuracyTreesM, &accuracy.TreesM, defaultTreesM},
		{envAccuracyBuildingsM, &accuracy.BuildingsM, defaultBuildingsM},
		{envAccuracyUnclearM, &accuracy.UnclearM, defaultUnclearM},
		{envAccuracyFallbackZoom, &accuracy.FallbackZoom, defaultFallbackZoom},
	}
	for _, number := range numbers {
		value, err := positiveNumberOrDefault(number.key, number.fallback)
		if err != nil {
			return err
		}

		*number.target = value
	}

	return nil
}

func (serviceConfig *Config) validate() error {
	required := []struct {
		key   string
		value string
	}{
		{envHTTPAddress, serviceConfig.HTTPAddress},
		{envMetricsAddress, serviceConfig.MetricsAddress},
		{envConsulAddress, serviceConfig.ConsulAddress},
		{envDatabaseURL, serviceConfig.DatabaseURL},
		{envRabbitMQURL, serviceConfig.RabbitMQURL},
		{envOIDCIssuer, serviceConfig.OIDCIssuer},
		{envOIDCJWKSURL, serviceConfig.OIDCJWKSURL},
		{envOIDCAudience, serviceConfig.OIDCAudience},
	}
	for _, setting := range required {
		if strings.TrimSpace(setting.value) == "" {
			return fmt.Errorf(formatMustNotBeEmpty, setting.key)
		}
	}

	return nil
}

func parsePort(address string) (int, error) {
	_, portValue, err := net.SplitHostPort(address)
	if err != nil {
		return 0, fmt.Errorf(wrapParse, envHTTPAddress, err)
	}

	port, err := strconv.Atoi(portValue)
	if err != nil || port < 1 || port > maximumPort {
		return 0, errInvalidPort
	}

	return port, nil
}

func defaultInstanceID() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf(wrapGetHostname, err)
	}

	return ServiceName + "-" + hostname, nil
}
