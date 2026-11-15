package domain

import "time"

type PlotEventType string

const (
	PlotCreated   PlotEventType = "plot.created"
	PlotUpdated   PlotEventType = "plot.updated"
	PlotCancelled PlotEventType = "plot.cancelled"
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

func (event PlotEvent) ChangesPlot() bool {
	return event.Type == PlotCreated || event.Type == PlotUpdated
}
