package handler

import (
	"time"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/handler/respond"
)

type coordinateDTO struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
}

type createPlotRequest struct {
	Name   string          `json:"name"`
	Points []coordinateDTO `json:"points"`
}

type updatePlotRequest struct {
	Name   *string          `json:"name"`
	Points *[]coordinateDTO `json:"points"`
}

type previewPlotRequest struct {
	Points *[]coordinateDTO `json:"points"`
}

type plotResponse struct {
	ID           string          `json:"id"`
	OwnerID      string          `json:"owner_id"`
	Name         string          `json:"name"`
	Points       []coordinateDTO `json:"points"`
	AreaM2       float64         `json:"area_m2"`
	AreaAres     float64         `json:"area_ares"`
	AreaHectares float64         `json:"area_hectares"`
	PerimeterM   float64         `json:"perimeter_m"`
	Version      int64           `json:"version"`
	Status       string          `json:"status"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type previewPlotResponse struct {
	AreaM2       *float64                  `json:"area_m2"`
	AreaAres     *float64                  `json:"area_ares"`
	AreaHectares *float64                  `json:"area_hectares"`
	PerimeterM   *float64                  `json:"perimeter_m"`
	Valid        bool                      `json:"valid"`
	Errors       []respond.ProblemResponse `json:"errors"`
}

func (body updatePlotRequest) toChanges() domain.PlotChanges {
	changes := domain.PlotChanges{Name: body.Name}
	if body.Points != nil {
		points := toCoordinates(*body.Points)
		changes.Points = &points
	}

	return changes
}

func toCoordinates(points []coordinateDTO) []domain.Coordinate {
	coordinates := make([]domain.Coordinate, len(points))
	for i, point := range points {
		coordinates[i] = domain.Coordinate{Latitude: point.Latitude, Longitude: point.Longitude}
	}

	return coordinates
}

func newPlotResponse(plot domain.Plot) plotResponse {
	points := make([]coordinateDTO, len(plot.Points))
	for i, point := range plot.Points {
		points[i] = coordinateDTO{Latitude: point.Latitude, Longitude: point.Longitude}
	}

	return plotResponse{
		ID:           plot.ID,
		OwnerID:      plot.OwnerID,
		Name:         plot.Name,
		Points:       points,
		AreaM2:       plot.AreaM2,
		AreaAres:     domain.SquareMetersToAres(plot.AreaM2),
		AreaHectares: domain.SquareMetersToHectares(plot.AreaM2),
		PerimeterM:   plot.PerimeterM,
		Version:      plot.Version,
		Status:       string(plot.Status),
		CreatedAt:    plot.CreatedAt,
		UpdatedAt:    plot.UpdatedAt,
	}
}

func newPreviewPlotResponse(inspection domain.PolygonInspection) previewPlotResponse {
	problems := make([]respond.ProblemResponse, 0, len(inspection.Errors))
	for _, problem := range inspection.Errors {
		problems = append(problems, respond.NewProblemResponse(problem))
	}

	if !inspection.Valid() {
		return previewPlotResponse{Errors: problems}
	}

	area := inspection.Area

	return previewPlotResponse{
		AreaM2:       &area.AreaM2,
		AreaAres:     new(domain.SquareMetersToAres(area.AreaM2)),
		AreaHectares: new(domain.SquareMetersToHectares(area.AreaM2)),
		PerimeterM:   &area.PerimeterM,
		Valid:        true,
		Errors:       problems,
	}
}
