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
	Plot       Plot
}

func NewPlotEvent(id string, eventType PlotEventType, plot Plot) PlotEvent {
	return PlotEvent{ID: id, Type: eventType, OccurredAt: plot.UpdatedAt, Plot: plot}
}
