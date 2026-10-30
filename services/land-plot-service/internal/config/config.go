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
	defaultConsulAddress   = "http://localhost:8500"
	defaultShutdownTimeout = 10 * time.Second
	defaultConsulTimeout   = 3 * time.Second
	defaultDatabaseTimeout = 5 * time.Second
	defaultRabbitMQTimeout = 5 * time.Second
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
	ServiceAddress  string
	InstanceID      string
	DatabaseURL     string
	DatabaseTimeout time.Duration
	JWTSecret       string
	RabbitMQURL     string
	RabbitMQTimeout time.Duration
}

func Load() (Config, error) {
	serviceConfig := Config{
		HTTPAddress:    stringOrDefault(envHTTPAddress, defaultHTTPAddress),
		ConsulAddress:  stringOrDefault(envConsulAddress, defaultConsulAddress),
		ServiceAddress: os.Getenv(envServiceAddress),
		InstanceID:     os.Getenv(envServiceInstanceID),
		DatabaseURL:    os.Getenv(envDatabaseURL),
		JWTSecret:      os.Getenv(envJWTSecret),
		RabbitMQURL:    os.Getenv(envRabbitMQURL),
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
		{envDatabaseTimeout, &serviceConfig.DatabaseTimeout, defaultDatabaseTimeout},
		{envRabbitMQTimeout, &serviceConfig.RabbitMQTimeout, defaultRabbitMQTimeout},
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
		{envDatabaseURL, serviceConfig.DatabaseURL},
		{envRabbitMQURL, serviceConfig.RabbitMQURL},
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
