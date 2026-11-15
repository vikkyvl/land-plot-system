package domain

import "time"

const (
	SquareMetersPerAre     = 100.0
	SquareMetersPerHectare = 10_000.0
)

type Report struct {
	ID           string
	OwnerID      string
	PlotID       string
	PlotName     string
	PlotVersion  int64
	AreaM2       float64
	AreaAres     float64
	AreaHectares float64
	PerimeterM   float64
	CreatedAt    time.Time
}

func NewReport(id string, plot PlotSnapshot, now time.Time) Report {
	return Report{
		ID:           id,
		OwnerID:      plot.OwnerID,
		PlotID:       plot.ID,
		PlotName:     plot.Name,
		PlotVersion:  plot.Version,
		AreaM2:       plot.AreaM2,
		AreaAres:     plot.AreaM2 / SquareMetersPerAre,
		AreaHectares: plot.AreaM2 / SquareMetersPerHectare,
		PerimeterM:   plot.PerimeterM,
		CreatedAt:    now.UTC(),
	}
}
