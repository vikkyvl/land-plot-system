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

const ServiceName = "report-service"

const (
	defaultHTTPAddress       = ":8083"
	defaultOIDCAudience      = "land-plot-api"
	defaultOIDCTimeout       = 5 * time.Second
	defaultConsulAddress     = "http://localhost:8500"
	defaultShutdownTimeout   = 10 * time.Second
	defaultConsulTimeout     = 3 * time.Second
	defaultDatabaseTimeout   = 5 * time.Second
	defaultUpstreamTimeout   = 5 * time.Second
	defaultRabbitMQTimeout   = 5 * time.Second
	defaultProcessingTimeout = 5 * time.Second
	defaultOutboxInterval    = time.Second
	defaultOutboxAttempts    = 10
	defaultOutboxCleanup     = time.Minute
	defaultOutboxRetention   = time.Hour
	defaultEventPrefetch     = 10
	defaultPlotServiceName   = "land-plot-service"
	maximumPort              = 65535
)

const (
	defaultElevationURL     = "https://api.open-meteo.com/v1/elevation"
	defaultOpenMeteoTimeout = 3 * time.Second
	defaultTerrainGridSide  = 10
	defaultTerrainMarginM   = 90
	defaultCoarseAreaM2     = 10_000
	minimumTerrainGridSide  = 2
	maximumTerrainGridSide  = 10
)

var (
	errInvalidGridSide = fmt.Errorf(
		formatMustBeBetween,
		envTerrainGridSide,
		minimumTerrainGridSide,
		maximumTerrainGridSide,
	)
	errInvalidPort     = errors.New(envHTTPAddress + " must contain a valid port")
	errInvalidPrefetch = errors.New(envRabbitMQPrefetch + " must be a positive integer")
)

type Config struct {
	HTTPAddress       string
	HTTPPort          int
	ShutdownTimeout   time.Duration
	ConsulAddress     string
	ConsulTimeout     time.Duration
	ServiceAddress    string
	InstanceID        string
	DatabaseURL       string
	DatabaseTimeout   time.Duration
	OIDCIssuer        string
	OIDCJWKSURL       string
	OIDCAudience      string
	OIDCTimeout       time.Duration
	UpstreamTimeout   time.Duration
	PlotServiceName   string
	RabbitMQURL       string
	RabbitMQTimeout   time.Duration
	ProcessingTimeout time.Duration
	EventPrefetch     int
	OutboxInterval    time.Duration
	FailNamePrefix    string
	OutboxAttempts    int

	OutboxCleanupInterval time.Duration
	OutboxRetention       time.Duration

	OpenMeteo OpenMeteoConfig
	Terrain   TerrainConfig
}

type OpenMeteoConfig struct {
	ElevationURL string
	Timeout      time.Duration
}

type TerrainConfig struct {
	GridSide     int
	MarginM      float64
	CoarseAreaM2 float64
}

func Load() (Config, error) {
	serviceConfig := Config{
		HTTPAddress:     stringOrDefault(envHTTPAddress, defaultHTTPAddress),
		ConsulAddress:   stringOrDefault(envConsulAddress, defaultConsulAddress),
		ServiceAddress:  os.Getenv(envServiceAddress),
		InstanceID:      os.Getenv(envServiceInstanceID),
		DatabaseURL:     os.Getenv(envDatabaseURL),
		OIDCIssuer:      os.Getenv(envOIDCIssuer),
		OIDCJWKSURL:     os.Getenv(envOIDCJWKSURL),
		OIDCAudience:    stringOrDefault(envOIDCAudience, defaultOIDCAudience),
		PlotServiceName: stringOrDefault(envLandPlotServiceName, defaultPlotServiceName),
		RabbitMQURL:     os.Getenv(envRabbitMQURL),
		FailNamePrefix:  os.Getenv(envReportFailNamePrefix),
	}
	if err := serviceConfig.loadDurations(); err != nil {
		return Config{}, err
	}

	if err := serviceConfig.validate(); err != nil {
		return Config{}, err
	}

	var err error
	if serviceConfig.EventPrefetch, err = parsePrefetch(); err != nil {
		return Config{}, err
	}

	if serviceConfig.OutboxAttempts, err = positiveIntOrDefault(
		envOutboxMaxAttempts,
		defaultOutboxAttempts,
	); err != nil {
		return Config{}, err
	}

	if serviceConfig.HTTPPort, err = parsePort(serviceConfig.HTTPAddress); err != nil {
		return Config{}, err
	}

	if err := serviceConfig.loadExternalData(); err != nil {
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
		{envOIDCTimeout, &serviceConfig.OIDCTimeout, defaultOIDCTimeout},
		{envConsulTimeout, &serviceConfig.ConsulTimeout, defaultConsulTimeout},
		{envDatabaseTimeout, &serviceConfig.DatabaseTimeout, defaultDatabaseTimeout},
		{envUpstreamTimeout, &serviceConfig.UpstreamTimeout, defaultUpstreamTimeout},
		{envRabbitMQTimeout, &serviceConfig.RabbitMQTimeout, defaultRabbitMQTimeout},
		{envMessageProcessingTimeout, &serviceConfig.ProcessingTimeout, defaultProcessingTimeout},
		{envOutboxPollInterval, &serviceConfig.OutboxInterval, defaultOutboxInterval},
		{envOutboxCleanupInterval, &serviceConfig.OutboxCleanupInterval, defaultOutboxCleanup},
		{envOutboxRetention, &serviceConfig.OutboxRetention, defaultOutboxRetention},
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

func (serviceConfig *Config) loadExternalData() error {
	serviceConfig.OpenMeteo.ElevationURL = stringOrDefault(envOpenMeteoElevationURL, defaultElevationURL)
	if err := requireHTTPURL(envOpenMeteoElevationURL, serviceConfig.OpenMeteo.ElevationURL); err != nil {
		return err
	}

	var err error
	if serviceConfig.OpenMeteo.Timeout, err = durationOrDefault(
		envOpenMeteoTimeout,
		defaultOpenMeteoTimeout,
	); err != nil {
		return err
	}

	terrain := &serviceConfig.Terrain
	if terrain.GridSide, err = positiveIntOrDefault(envTerrainGridSide, defaultTerrainGridSide); err != nil {
		return err
	}

	if terrain.GridSide < minimumTerrainGridSide || terrain.GridSide > maximumTerrainGridSide {
		return errInvalidGridSide
	}

	if terrain.MarginM, err = positiveNumberOrDefault(envTerrainMarginM, defaultTerrainMarginM); err != nil {
		return err
	}

	terrain.CoarseAreaM2, err = positiveNumberOrDefault(envTerrainCoarseAreaM2, defaultCoarseAreaM2)

	return err
}

func (serviceConfig *Config) validate() error {
	required := []struct {
		key   string
		value string
	}{
		{envHTTPAddress, serviceConfig.HTTPAddress},
		{envConsulAddress, serviceConfig.ConsulAddress},
		{envOIDCIssuer, serviceConfig.OIDCIssuer},
		{envOIDCJWKSURL, serviceConfig.OIDCJWKSURL},
		{envOIDCAudience, serviceConfig.OIDCAudience},
		{envDatabaseURL, serviceConfig.DatabaseURL},
		{envLandPlotServiceName, serviceConfig.PlotServiceName},
		{envRabbitMQURL, serviceConfig.RabbitMQURL},
	}
	for _, setting := range required {
		if strings.TrimSpace(setting.value) == "" {
			return fmt.Errorf(formatMustNotBeEmpty, setting.key)
		}
	}

	return nil
}

func parsePrefetch() (int, error) {
	value, ok := os.LookupEnv(envRabbitMQPrefetch)
	if !ok {
		return defaultEventPrefetch, nil
	}

	prefetch, err := strconv.Atoi(value)
	if err != nil || prefetch < 1 {
		return 0, errInvalidPrefetch
	}

	return prefetch, nil
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
