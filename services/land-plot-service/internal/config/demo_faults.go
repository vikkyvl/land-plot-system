package config

import (
	"fmt"
	"os"
	"strconv"
)

func demoFaultsEnabled() (bool, error) {
	value, ok := os.LookupEnv(envDemoFaultsEnabled)
	if !ok || value == "" {
		return false, nil
	}

	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf(formatMustBeABoolean, envDemoFaultsEnabled)
	}

	return enabled, nil
}
