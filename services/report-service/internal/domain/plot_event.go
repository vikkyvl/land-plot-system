package domain

import "time"

type PlotEventType string

const (
	PlotCreated PlotEventType = "plot.created"
	PlotUpdated PlotEventType = "plot.updated"
)

type PlotEvent struct {
	ID         string
	Type       PlotEventType
	OccurredAt time.Time
	PlotID     string
	OwnerID    string
	Name       string
	AreaM2     float64
	Version    int64
}
