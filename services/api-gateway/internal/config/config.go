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

const ServiceName = "api-gateway"

const (
	defaultHTTPAddress        = ":8080"
	defaultMetricsAddress     = ":9464"
	defaultCORSAllowedOrigins = "https://cad.kpi.ua,http://localhost:5173,http://localhost:4173"
	defaultOIDCAudience       = "land-plot-api"
	defaultOIDCTimeout        = 5 * time.Second
	defaultConsulAddress      = "http://localhost:8500"
	defaultShutdownTimeout    = 10 * time.Second
	defaultConsulTimeout      = 3 * time.Second
	defaultUpstreamTimeout    = 5 * time.Second
	defaultPreviewRateLimit   = 60
	maximumPort               = 65535
)

var errInvalidPort = errors.New(envHTTPAddress + " must contain a valid port")

type Config struct {
	HTTPAddress               string
	HTTPPort                  int
	MetricsAddress            string
	OTLPEndpoint              string
	ShutdownTimeout           time.Duration
	ConsulAddress             string
	ConsulTimeout             time.Duration
	UpstreamTimeout           time.Duration
	ServiceAddress            string
	InstanceID                string
	OIDCIssuer                string
	OIDCJWKSURL               string
	OIDCAudience              string
	OIDCTimeout               time.Duration
	AllowedOrigins            []string
	PreviewRateLimitPerMinute int
}

func Load() (Config, error) {
	serviceConfig := Config{
		HTTPAddress:    stringOrDefault(envHTTPAddress, defaultHTTPAddress),
		MetricsAddress: stringOrDefault(envMetricsAddress, defaultMetricsAddress),
		OTLPEndpoint:   os.Getenv(envOTLPEndpoint),
		ConsulAddress:  stringOrDefault(envConsulAddress, defaultConsulAddress),
		ServiceAddress: os.Getenv(envServiceAddress),
		InstanceID:     os.Getenv(envServiceInstanceID),
		OIDCIssuer:     os.Getenv(envOIDCIssuer),
		OIDCJWKSURL:    os.Getenv(envOIDCJWKSURL),
		OIDCAudience:   stringOrDefault(envOIDCAudience, defaultOIDCAudience),
	}
	if err := serviceConfig.loadDurations(); err != nil {
		return Config{}, err
	}

	if err := serviceConfig.validate(); err != nil {
		return Config{}, err
	}

	var err error
	if serviceConfig.AllowedOrigins, err = originsOrDefault(
		envCORSAllowedOrigins,
		defaultCORSAllowedOrigins,
	); err != nil {
		return Config{}, err
	}

	if serviceConfig.PreviewRateLimitPerMinute, err = positiveIntOrDefault(
		envPreviewRateLimit,
		defaultPreviewRateLimit,
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
		{envOIDCTimeout, &serviceConfig.OIDCTimeout, defaultOIDCTimeout},
		{envConsulTimeout, &serviceConfig.ConsulTimeout, defaultConsulTimeout},
		{envUpstreamTimeout, &serviceConfig.UpstreamTimeout, defaultUpstreamTimeout},
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

func (serviceConfig *Config) validate() error {
	required := []struct {
		key   string
		value string
	}{
		{envHTTPAddress, serviceConfig.HTTPAddress},
		{envMetricsAddress, serviceConfig.MetricsAddress},
		{envConsulAddress, serviceConfig.ConsulAddress},
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
