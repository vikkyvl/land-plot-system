package domain

import "time"

type PlotVersion struct {
	PlotID     string
	Version    int64
	Name       string
	Points     []Coordinate
	AreaM2     float64
	PerimeterM float64
	CreatedAt  time.Time
}

func (plot Plot) ConfirmedVersion() PlotVersion {
	return PlotVersion{
		PlotID:     plot.ID,
		Version:    plot.Version,
		Name:       plot.Name,
		Points:     clonePoints(plot.Points),
		AreaM2:     plot.AreaM2,
		PerimeterM: plot.PerimeterM,
		CreatedAt:  plot.UpdatedAt,
	}
}
