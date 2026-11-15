package postgres

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

func TestSummaryAppliedMatchesContractV1(t *testing.T) {
	payload, err := encodeSummaryApplied(domain.SummaryApplied{
		ID:             "reply-1",
		AppliedEventID: "event-1",
		PlotID:         "plot-1",
		PlotVersion:    2,
		OccurredAt:     time.Date(2026, time.November, 8, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("encodeSummaryApplied() error = %v", err)
	}

	var fields map[string]any
	if err := json.Unmarshal(payload, &fields); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	want := map[string]any{
		"schema_version":   float64(1),
		"event_id":         "reply-1",
		"event_type":       "plot.summary.applied",
		"occurred_at":      "2026-11-08T10:00:00Z",
		"applied_event_id": "event-1",
		"plot_id":          "plot-1",
		"plot_version":     float64(2),
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
