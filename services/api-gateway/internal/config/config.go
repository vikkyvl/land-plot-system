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
	defaultHTTPAddress     = ":8080"
	defaultConsulAddress   = "http://localhost:8500"
	defaultShutdownTimeout = 10 * time.Second
	defaultConsulTimeout   = 3 * time.Second
	defaultUpstreamTimeout = 5 * time.Second
	minimumJWTSecretLength = 32
	maximumPort            = 65535
)

var errInvalidPort = errors.New(envHTTPAddress + " must contain a valid port")

type Config struct {
	HTTPAddress     string
	HTTPPort        int
	ShutdownTimeout time.Duration
	ConsulAddress   string
	ConsulTimeout   time.Duration
	UpstreamTimeout time.Duration
	ServiceAddress  string
	InstanceID      string
	JWTSecret       string
}

func Load() (Config, error) {
	serviceConfig := Config{
		HTTPAddress:    stringOrDefault(envHTTPAddress, defaultHTTPAddress),
		ConsulAddress:  stringOrDefault(envConsulAddress, defaultConsulAddress),
		ServiceAddress: os.Getenv(envServiceAddress),
		InstanceID:     os.Getenv(envServiceInstanceID),
		JWTSecret:      os.Getenv(envJWTSecret),
	}
	if err := serviceConfig.loadDurations(); err != nil {
		return Config{}, err
	}

	if err := serviceConfig.validate(); err != nil {
		return Config{}, err
	}

	var err error
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
		{envConsulAddress, serviceConfig.ConsulAddress},
	}
	for _, setting := range required {
		if strings.TrimSpace(setting.value) == "" {
			return fmt.Errorf(formatMustNotBeEmpty, setting.key)
		}
	}

	if len(strings.TrimSpace(serviceConfig.JWTSecret)) < minimumJWTSecretLength {
		return fmt.Errorf(formatMustContainAtLeastCharacters, envJWTSecret, minimumJWTSecretLength)
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
