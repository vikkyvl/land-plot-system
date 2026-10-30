package rabbitmq

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

const (
	plotMessageSchemaV1 = 1
	maximumMessageBytes = 1 << 20
)

type plotMessage struct {
	SchemaVersion int       `json:"schema_version"`
	EventID       string    `json:"event_id"`
	EventType     string    `json:"event_type"`
	OccurredAt    time.Time `json:"occurred_at"`
	PlotID        string    `json:"plot_id"`
	OwnerID       string    `json:"owner_id"`
	Name          string    `json:"name"`
	AreaM2        float64   `json:"area_m2"`
	Version       int64     `json:"version"`
}

func decodePlotMessage(body []byte, routingKey string) (domain.PlotEvent, error) {
	if len(body) == 0 || len(body) > maximumMessageBytes {
		return domain.PlotEvent{}, fmt.Errorf(wrapInvalidBodySize, ErrInvalidMessage)
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	var message plotMessage
	if err := decoder.Decode(&message); err != nil {
		return domain.PlotEvent{}, fmt.Errorf(wrapDecodeJSON, ErrInvalidMessage, err)
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return domain.PlotEvent{}, fmt.Errorf(wrapBodyMustContainOneJSONObject, ErrInvalidMessage)
	}

	if err := message.validate(); err != nil {
		return domain.PlotEvent{}, fmt.Errorf("%w: %w", ErrInvalidMessage, err)
	}

	if message.EventType != routingKey {
		return domain.PlotEvent{}, fmt.Errorf(wrapEventTypeDoesNotMatchRoutingKey, ErrInvalidMessage)
	}

	return message.toDomain(), nil
}

func (message plotMessage) validate() error {
	switch {
	case message.SchemaVersion != plotMessageSchemaV1:
		return errUnsupportedSchemaVersion
	case message.EventType != string(domain.PlotCreated) && message.EventType != string(domain.PlotUpdated):
		return errUnsupportedEventType
	case !isUUID(message.EventID):
		return errInvalidEventID
	case !isUUID(message.PlotID):
		return errInvalidPlotID
	case !isUUID(message.OwnerID):
		return errInvalidOwnerID
	case message.OccurredAt.IsZero():
		return errMissingOccurrenceTime
	case strings.TrimSpace(message.Name) == "":
		return errMissingPlotName
	case math.IsNaN(message.AreaM2) || math.IsInf(message.AreaM2, 0) || message.AreaM2 <= 0:
		return errInvalidArea
	case message.Version < 1:
		return errInvalidAggregateVersion
	}

	return nil
}

func (message plotMessage) toDomain() domain.PlotEvent {
	return domain.PlotEvent{
		ID:         message.EventID,
		Type:       domain.PlotEventType(message.EventType),
		OccurredAt: message.OccurredAt,
		PlotID:     message.PlotID,
		OwnerID:    message.OwnerID,
		Name:       message.Name,
		AreaM2:     message.AreaM2,
		Version:    message.Version,
	}
}

func isUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}
