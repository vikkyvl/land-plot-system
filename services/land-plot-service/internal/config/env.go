package config

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"time"
)

func stringOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
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

func positiveNumberOrDefault(key string, fallback float64) (float64, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || !(parsed > 0) || math.IsInf(parsed, 0) {
		return 0, fmt.Errorf(formatMustBeAPositiveNumber, key)
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
