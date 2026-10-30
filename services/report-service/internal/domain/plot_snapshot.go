package domain

type PlotSnapshot struct {
	ID         string
	OwnerID    string
	Name       string
	AreaM2     float64
	PerimeterM float64
	Version    int64
}
