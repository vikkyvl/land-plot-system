package postgres

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

const plotMessageSchemaV1 = 1

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

func newPlotMessage(event domain.PlotEvent) plotMessage {
	return plotMessage{
		SchemaVersion: plotMessageSchemaV1,
		EventID:       event.ID,
		EventType:     string(event.Type),
		OccurredAt:    event.OccurredAt,
		PlotID:        event.Plot.ID,
		OwnerID:       event.Plot.OwnerID,
		Name:          event.Plot.Name,
		AreaM2:        event.Plot.AreaM2,
		Version:       event.Plot.Version,
	}
}

func encodePlotEvent(event domain.PlotEvent) ([]byte, error) {
	payload, err := json.Marshal(newPlotMessage(event))
	if err != nil {
		return nil, fmt.Errorf(wrapEncodePlotEvent, err)
	}

	return payload, nil
}
