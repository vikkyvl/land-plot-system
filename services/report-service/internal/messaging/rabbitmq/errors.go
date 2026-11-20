package rabbitmq

import (
	"errors"
	"fmt"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

var ErrInvalidMessage = errors.New("invalid plot event")

var errDeliveriesClosed = errors.New("plot event delivery channel closed")

var (
	errBrokerRejectedMessage = fmt.Errorf(wrapBrokerRejectedMessage, domain.ErrMessageRejected)
	errMessageUnroutable     = fmt.Errorf(wrapMessageUnroutable, domain.ErrMessageRejected)
)

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
