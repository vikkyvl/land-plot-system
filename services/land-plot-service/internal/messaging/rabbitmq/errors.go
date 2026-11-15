package rabbitmq

import (
	"errors"
	"fmt"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

var (
	errDeliveriesClosed         = errors.New("saga delivery channel closed")
	errInvalidEventID           = errors.New("invalid event ID")
	errInvalidSagaMessage       = errors.New("invalid saga message")
	errUnsupportedEventType     = errors.New("unsupported event type")
	errUnsupportedSchemaVersion = errors.New("unsupported schema version")
)

var (
	errBrokerRejectedMessage = fmt.Errorf(wrapBrokerRejectedMessage, domain.ErrMessageRejected)
	errMessageUnroutable     = fmt.Errorf(wrapMessageUnroutable, domain.ErrMessageRejected)
)
