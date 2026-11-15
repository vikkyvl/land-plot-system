package handler

import (
	"time"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

type createReportRequest struct {
	PlotID string `json:"plot_id"`
}

type reportResponse struct {
	ID           string    `json:"id"`
	PlotID       string    `json:"plot_id"`
	PlotName     string    `json:"plot_name"`
	PlotVersion  int64     `json:"plot_version"`
	AreaM2       float64   `json:"area_m2"`
	AreaAres     float64   `json:"area_ares"`
	AreaHectares float64   `json:"area_hectares"`
	PerimeterM   float64   `json:"perimeter_m"`
	CreatedAt    time.Time `json:"created_at"`
}

func newReportResponse(report domain.Report) reportResponse {
	return reportResponse{
		ID:           report.ID,
		PlotID:       report.PlotID,
		PlotName:     report.PlotName,
		PlotVersion:  report.PlotVersion,
		AreaM2:       report.AreaM2,
		AreaAres:     report.AreaAres,
		AreaHectares: report.AreaHectares,
		PerimeterM:   report.PerimeterM,
		CreatedAt:    report.CreatedAt,
	}
}
