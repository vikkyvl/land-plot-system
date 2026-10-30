package rabbitmq

import "errors"

var ErrInvalidMessage = errors.New("invalid plot event")

var (
	errInvalidAggregateVersion  = errors.New("invalid aggregate version")
	errInvalidArea              = errors.New("invalid area")
	errInvalidEventID           = errors.New("invalid event ID")
	errInvalidOwnerID           = errors.New("invalid owner ID")
	errInvalidPlotID            = errors.New("invalid plot ID")
	errMissingOccurrenceTime    = errors.New("missing occurrence time")
	errMissingPlotName          = errors.New("missing plot name")
	errUnsupportedEventType     = errors.New("unsupported event type")
	errUnsupportedSchemaVersion = errors.New("unsupported schema version")
)
