package domain

const PlotStatusConfirmed = "CONFIRMED"

type PlotSnapshot struct {
	ID         string
	OwnerID    string
	Name       string
	AreaM2     float64
	PerimeterM float64
	Version    int64
	Status     string
	Points     []Coordinate
	Accuracy   *AreaAccuracy
}

func (plot PlotSnapshot) IsConfirmed() bool {
	return plot.Status == PlotStatusConfirmed
}
