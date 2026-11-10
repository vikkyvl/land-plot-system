package domain

import "time"

const SummaryAppliedEvent = "plot.summary.applied"

type SummaryApplied struct {
	ID             string
	AppliedEventID string
	PlotID         string
	PlotVersion    int64
	OccurredAt     time.Time
}

type OutboxMessage struct {
	ID         string
	RoutingKey string
	Payload    []byte
}

func NewSummaryApplied(id string, applied PlotEvent, now time.Time) SummaryApplied {
	return SummaryApplied{
		ID:             id,
		AppliedEventID: applied.ID,
		PlotID:         applied.PlotID,
		PlotVersion:    applied.Version,
		OccurredAt:     now.UTC(),
	}
}
