package rabbitmq

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

func TestPlotMessageMatchesContractV1(t *testing.T) {
	occurredAt := time.Date(2026, time.October, 26, 12, 0, 0, 0, time.UTC)
	event := domain.PlotEvent{
		ID:         "event-1",
		Type:       domain.PlotUpdated,
		OccurredAt: occurredAt,
		Plot:       domain.Plot{ID: "plot-1", OwnerID: "owner-1", Name: "Plot", AreaM2: 100, Version: 2},
	}

	encoded, err := json.Marshal(newPlotMessage(event))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	want := map[string]any{
		"schema_version": float64(1),
		"event_id":       "event-1",
		"event_type":     "plot.updated",
		"occurred_at":    "2026-10-26T12:00:00Z",
		"plot_id":        "plot-1",
		"owner_id":       "owner-1",
		"name":           "Plot",
		"area_m2":        float64(100),
		"version":        float64(2),
	}
	if len(fields) != len(want) {
		t.Fatalf("fields = %v, want %v", fields, want)
	}

	for key, value := range want {
		if fields[key] != value {
			t.Fatalf("%s = %v, want %v", key, fields[key], value)
		}
	}
}
