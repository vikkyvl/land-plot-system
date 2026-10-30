package httpclient

import "github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"

type plotResponse struct {
	ID         string  `json:"id"`
	OwnerID    string  `json:"owner_id"`
	Name       string  `json:"name"`
	AreaM2     float64 `json:"area_m2"`
	PerimeterM float64 `json:"perimeter_m"`
	Version    int64   `json:"version"`
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
		plot.AreaM2 > 0 && plot.PerimeterM > 0 && plot.Version >= 1
}

func (plot plotResponse) toDomain() domain.PlotSnapshot {
	return domain.PlotSnapshot{
		ID:         plot.ID,
		OwnerID:    plot.OwnerID,
		Name:       plot.Name,
		AreaM2:     plot.AreaM2,
		PerimeterM: plot.PerimeterM,
		Version:    plot.Version,
	}
}

func (trace traceResponse) toDomain() []domain.TraceHop {
	hops := make([]domain.TraceHop, len(trace.Path))
	for i, hop := range trace.Path {
		hops[i] = domain.TraceHop{Service: hop.Service, Instance: hop.Instance, Protocol: hop.Protocol}
	}

	return hops
}
