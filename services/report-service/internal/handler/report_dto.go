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
	PlotDeleted  bool      `json:"plot_deleted"`

	Accuracy *accuracyResponse `json:"accuracy"`
	Terrain  *terrainResponse  `json:"terrain"`
	Climate  *climateResponse  `json:"climate"`

	Boundary              []boundaryPointResponse `json:"boundary"`
	GeneratedFrom         generatedFromResponse   `json:"generated_from"`
	LatestReportID        *string                 `json:"latest_report_id"`
	LatestReportCreatedAt *time.Time              `json:"latest_report_created_at"`
}

type boundaryPointResponse struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
}

type generatedFromResponse struct {
	ClimateFirstYear *int       `json:"climate_first_year"`
	ClimateLastYear  *int       `json:"climate_last_year"`
	TerrainAt        *time.Time `json:"terrain_at"`
}

func newReportResponse(report domain.Report) reportResponse {
	response := reportResponse{
		ID:            report.ID,
		PlotID:        report.PlotID,
		PlotName:      report.PlotName,
		PlotVersion:   report.PlotVersion,
		AreaM2:        report.AreaM2,
		AreaAres:      report.AreaAres,
		AreaHectares:  report.AreaHectares,
		PerimeterM:    report.PerimeterM,
		CreatedAt:     report.CreatedAt,
		PlotDeleted:   report.PlotDeleted(),
		Accuracy:      newAccuracyResponse(report.Accuracy),
		Terrain:       newTerrainResponse(report.Terrain),
		Climate:       newClimateResponse(report.Climate),
		Boundary:      newBoundaryResponse(report.Boundary),
		GeneratedFrom: newGeneratedFromResponse(report),
	}
	if report.Latest != nil {
		response.LatestReportID = &report.Latest.ID
		response.LatestReportCreatedAt = &report.Latest.CreatedAt
	}

	return response
}

func newBoundaryResponse(points []domain.Coordinate) []boundaryPointResponse {
	boundary := make([]boundaryPointResponse, len(points))
	for index, point := range points {
		boundary[index] = boundaryPointResponse{Latitude: point.Latitude, Longitude: point.Longitude}
	}

	return boundary
}

func newGeneratedFromResponse(report domain.Report) generatedFromResponse {
	var generated generatedFromResponse

	if report.Climate != nil && report.Climate.Status == domain.DataAvailable {
		generated.ClimateFirstYear = &report.Climate.FirstYear
		generated.ClimateLastYear = &report.Climate.LastYear
	}

	if report.Terrain != nil && report.Terrain.Status == domain.DataAvailable {
		generated.TerrainAt = &report.CreatedAt
	}

	return generated
}
