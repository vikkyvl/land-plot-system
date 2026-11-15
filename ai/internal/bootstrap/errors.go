package bootstrap

import "errors"

var (
	errMissingQuery         = errors.New("query is required")
	errUnknownTransport     = errors.New("transport must be stdio or http")
	errMissingCallArguments = errors.New("agent URL and task text are required")
	errInvalidAgentURL      = errors.New("agent URL must be an absolute http or https URL")
)
