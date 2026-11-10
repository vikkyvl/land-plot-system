package rabbitmq

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

func validPlotMessage(eventType domain.PlotEventType) plotMessage {
	return plotMessage{
		SchemaVersion: plotMessageSchemaV1,
		EventID:       uuid.NewString(),
		EventType:     string(eventType),
		OccurredAt:    time.Date(2026, time.October, 26, 12, 0, 0, 0, time.UTC),
		PlotID:        uuid.NewString(),
		OwnerID:       uuid.NewString(),
		Name:          "North field",
		AreaM2:        10_000,
		Version:       1,
	}
}

func encodeMessage(t *testing.T, message any) []byte {
	t.Helper()

	body, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	return body
}

func TestDecodePlotMessage(t *testing.T) {
	message := validPlotMessage(domain.PlotCreated)

	event, err := decodePlotMessage(encodeMessage(t, message), message.EventType)
	if err != nil {
		t.Fatalf("decodePlotMessage() error = %v", err)
	}

	if event.ID != message.EventID || event.Type != domain.PlotCreated || event.PlotID != message.PlotID ||
		event.Version != 1 {
		t.Fatalf("event = %#v", event)
	}
}

func TestDecodePlotMessageRejectsInvalidContract(t *testing.T) {
	tests := map[string]func(*plotMessage){
		"schema version": func(message *plotMessage) { message.SchemaVersion++ },
		"event type":     func(message *plotMessage) { message.EventType = "plot.deleted" },
		"event ID":       func(message *plotMessage) { message.EventID = "not-a-uuid" },
		"name":           func(message *plotMessage) { message.Name = " " },
		"area":           func(message *plotMessage) { message.AreaM2 = 0 },
		"version":        func(message *plotMessage) { message.Version = 0 },
	}
	for name, corrupt := range tests {
		t.Run(name, func(t *testing.T) {
			message := validPlotMessage(domain.PlotCreated)
			routingKey := message.EventType
			corrupt(&message)

			_, err := decodePlotMessage(encodeMessage(t, message), routingKey)
			if !errors.Is(err, ErrInvalidMessage) {
				t.Fatalf("decodePlotMessage() error = %v, want ErrInvalidMessage", err)
			}
		})
	}
}

func TestDecodePlotMessageRejectsUnknownFields(t *testing.T) {
	message := validPlotMessage(domain.PlotCreated)
	body := encodeMessage(t, message)
	body = append(body[:len(body)-1], []byte(`,"unexpected":true}`)...)

	if _, err := decodePlotMessage(body, message.EventType); !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("decodePlotMessage() error = %v, want ErrInvalidMessage", err)
	}
}

func TestDecodePlotMessageRejectsMismatchedRoutingKey(t *testing.T) {
	message := validPlotMessage(domain.PlotCreated)

	if _, err := decodePlotMessage(
		encodeMessage(t, message),
		string(domain.PlotUpdated),
	); !errors.Is(
		err,
		ErrInvalidMessage,
	) {
		t.Fatalf("decodePlotMessage() error = %v, want ErrInvalidMessage", err)
	}
}
