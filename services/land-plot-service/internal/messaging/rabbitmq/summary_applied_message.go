package rabbitmq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type summaryAppliedMessage struct {
	SchemaVersion  int       `json:"schema_version"`
	EventID        string    `json:"event_id"`
	EventType      string    `json:"event_type"`
	OccurredAt     time.Time `json:"occurred_at"`
	AppliedEventID string    `json:"applied_event_id"`
	PlotID         string    `json:"plot_id"`
	PlotVersion    int64     `json:"plot_version"`
}

func decodeSummaryApplied(body []byte) (summaryAppliedMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	var message summaryAppliedMessage
	if err := decoder.Decode(&message); err != nil {
		return summaryAppliedMessage{}, fmt.Errorf(wrapDecodeJSON, errInvalidSagaMessage, err)
	}

	switch {
	case message.SchemaVersion != summaryAppliedSchemaV1:
		return summaryAppliedMessage{}, fmt.Errorf("%w: %w", errInvalidSagaMessage, errUnsupportedSchemaVersion)
	case message.EventType != summaryAppliedEvent:
		return summaryAppliedMessage{}, fmt.Errorf("%w: %w", errInvalidSagaMessage, errUnsupportedEventType)
	case !isUUID(message.AppliedEventID):
		return summaryAppliedMessage{}, fmt.Errorf("%w: %w", errInvalidSagaMessage, errInvalidEventID)
	}

	return message, nil
}

func isUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}
