package httpclient

import "github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"

const minimumBoundaryPoints = 3

type plotResponse struct {
	ID         string  `json:"id"`
	OwnerID    string  `json:"owner_id"`
	Name       string  `json:"name"`
	AreaM2     float64 `json:"area_m2"`
	PerimeterM float64 `json:"perimeter_m"`
	Version    int64   `json:"version"`
	Status     string  `json:"status"`

	Points   []pointResponse   `json:"points"`
	Accuracy *accuracyResponse `json:"accuracy"`
}

type pointResponse struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
}

type accuracyResponse struct {
	AreaM2Margin       float64               `json:"area_m2_margin"`
	AreaHectaresMargin float64               `json:"area_hectares_margin"`
	Percent            float64               `json:"percent"`
	Confidence         float64               `json:"confidence"`
	Contributions      contributionsResponse `json:"contributions"`
	WorstSide          *int                  `json:"worst_side"`
	GroundShiftM       float64               `json:"ground_shift_m"`
	Basis              string                `json:"basis"`
}

type contributionsResponse struct {
	Scale        float64 `json:"scale"`
	Obstructions float64 `json:"obstructions"`
}

type traceHopResponse struct {
	Service  string `json:"service"`
	Instance string `json:"instance"`
	Protocol string `json:"protocol"`
}

type traceResponse struct {
	Path []traceHopResponse `json:"path"`
}

func (plot plotResponse) isValidFor(requestedID string) bool {
	return plot.ID == requestedID && plot.OwnerID != "" && plot.Name != "" &&
		plot.AreaM2 > 0 && plot.PerimeterM > 0 && plot.Version >= 1 && plot.Status != "" &&
		len(plot.Points) >= minimumBoundaryPoints
}

func (plot plotResponse) toDomain() domain.PlotSnapshot {
	return domain.PlotSnapshot{
		ID:         plot.ID,
		OwnerID:    plot.OwnerID,
		Name:       plot.Name,
		AreaM2:     plot.AreaM2,
		PerimeterM: plot.PerimeterM,
		Version:    plot.Version,
		Status:     plot.Status,
		Points:     toCoordinates(plot.Points),
		Accuracy:   plot.Accuracy.toDomain(),
	}
}

func toCoordinates(points []pointResponse) []domain.Coordinate {
	coordinates := make([]domain.Coordinate, len(points))
	for i, point := range points {
		coordinates[i] = domain.Coordinate{Latitude: point.Latitude, Longitude: point.Longitude}
	}

	return coordinates
}

func (accuracy *accuracyResponse) toDomain() *domain.AreaAccuracy {
	if accuracy == nil {
		return nil
	}

	return &domain.AreaAccuracy{
		AreaMarginM2:       accuracy.AreaM2Margin,
		AreaMarginHectares: accuracy.AreaHectaresMargin,
		Percent:            accuracy.Percent,
		Confidence:         accuracy.Confidence,
		ScaleShare:         accuracy.Contributions.Scale,
		ObstructionShare:   accuracy.Contributions.Obstructions,
		WorstSide:          accuracy.WorstSide,
		GroundShiftM:       accuracy.GroundShiftM,
		Basis:              accuracy.Basis,
	}
}

func (trace traceResponse) toDomain() []domain.TraceHop {
	hops := make([]domain.TraceHop, len(trace.Path))
	for i, hop := range trace.Path {
		hops[i] = domain.TraceHop{Service: hop.Service, Instance: hop.Instance, Protocol: hop.Protocol}
	}

	return hops
}

type errorResponse struct {
	Code string `json:"code"`
}
