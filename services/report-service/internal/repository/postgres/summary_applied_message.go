package postgres

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

const summaryAppliedSchemaV1 = 1

type summaryAppliedMessage struct {
	SchemaVersion  int       `json:"schema_version"`
	EventID        string    `json:"event_id"`
	EventType      string    `json:"event_type"`
	OccurredAt     time.Time `json:"occurred_at"`
	AppliedEventID string    `json:"applied_event_id"`
	PlotID         string    `json:"plot_id"`
	PlotVersion    int64     `json:"plot_version"`
}

func encodeSummaryApplied(reply domain.SummaryApplied) ([]byte, error) {
	payload, err := json.Marshal(summaryAppliedMessage{
		SchemaVersion:  summaryAppliedSchemaV1,
		EventID:        reply.ID,
		EventType:      domain.SummaryAppliedEvent,
		OccurredAt:     reply.OccurredAt,
		AppliedEventID: reply.AppliedEventID,
		PlotID:         reply.PlotID,
		PlotVersion:    reply.PlotVersion,
	})
	if err != nil {
		return nil, fmt.Errorf(wrapEncodeSummaryApplied, err)
	}

	return payload, nil
}
