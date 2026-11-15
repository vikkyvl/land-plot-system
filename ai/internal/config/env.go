package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

const maximumPort = 65535

type durationSetting struct {
	key      string
	target   *time.Duration
	fallback time.Duration
}

type intSetting struct {
	key      string
	target   *int
	fallback int
}

func stringOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return strings.TrimSpace(value)
	}

	return fallback
}

func requiredString(key string) (string, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", fmt.Errorf(formatMustNotBeEmpty, key)
	}

	return value, nil
}

func positiveIntOrDefault(key string, fallback int) (int, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0, fmt.Errorf(formatMustBeAPositiveInteger, key)
	}

	return parsed, nil
}

func durationOrDefault(key string, fallback time.Duration) (time.Duration, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf(formatMustBeAPositiveDuration, key)
	}

	return parsed, nil
}

func loadDurations(settings []durationSetting) error {
	for _, setting := range settings {
		value, err := durationOrDefault(setting.key, setting.fallback)
		if err != nil {
			return err
		}

		*setting.target = value
	}

	return nil
}

func loadInts(settings []intSetting) error {
	for _, setting := range settings {
		value, err := positiveIntOrDefault(setting.key, setting.fallback)
		if err != nil {
			return err
		}

		*setting.target = value
	}

	return nil
}

func httpURLOrDefault(key, fallback string) (string, error) {
	value := strings.TrimRight(stringOrDefault(key, fallback), "/")

	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", fmt.Errorf(formatMustBeAnHTTPURL, key)
	}

	return value, nil
}

func listenAddressOrDefault(key, fallback string) (string, error) {
	address := stringOrDefault(key, fallback)

	_, portValue, err := net.SplitHostPort(address)
	if err != nil {
		return "", fmt.Errorf(formatMustContainAPort, key)
	}

	port, err := strconv.Atoi(portValue)
	if err != nil || port < 1 || port > maximumPort {
		return "", fmt.Errorf(formatMustContainAPort, key)
	}

	return address, nil
}

func oneOfOrDefault(key, fallback string, allowed []string) (string, error) {
	value := stringOrDefault(key, fallback)
	if !slices.Contains(allowed, value) {
		return "", fmt.Errorf(formatMustBeOneOf, key, strings.Join(allowed, ", "))
	}

	return value, nil
}
