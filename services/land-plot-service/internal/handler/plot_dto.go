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

type captureDTO struct {
	Zoom    float64 `json:"zoom"`
	Layer   string  `json:"layer"`
	Pointer string  `json:"pointer"`
}

type obstructionDTO struct {
	Side   int    `json:"side"`
	Reason string `json:"reason"`
}

type contributionsDTO struct {
	Scale        float64 `json:"scale"`
	Obstructions float64 `json:"obstructions"`
}

type accuracyDTO struct {
	AreaM2Margin       float64          `json:"area_m2_margin"`
	AreaHectaresMargin float64          `json:"area_hectares_margin"`
	Percent            float64          `json:"percent"`
	Confidence         float64          `json:"confidence"`
	Contributions      contributionsDTO `json:"contributions"`
	WorstSide          *int             `json:"worst_side"`
	GroundShiftM       float64          `json:"ground_shift_m"`
	Basis              string           `json:"basis"`
}

type surveyRequest struct {
	Capture      *captureDTO      `json:"capture"`
	Obstructions []obstructionDTO `json:"obstructions"`
}

type createPlotRequest struct {
	surveyRequest

	Name   string          `json:"name"`
	Points []coordinateDTO `json:"points"`
}

type updatePlotRequest struct {
	Name         *string           `json:"name"`
	Points       *[]coordinateDTO  `json:"points"`
	Capture      *captureDTO       `json:"capture"`
	Obstructions *[]obstructionDTO `json:"obstructions"`
}

type previewPlotRequest struct {
	surveyRequest

	Points *[]coordinateDTO `json:"points"`
}

type plotResponse struct {
	ID           string                 `json:"id"`
	OwnerID      string                 `json:"owner_id"`
	Name         string                 `json:"name"`
	Points       []coordinateDTO        `json:"points"`
	AreaM2       float64                `json:"area_m2"`
	AreaAres     float64                `json:"area_ares"`
	AreaHectares float64                `json:"area_hectares"`
	PerimeterM   float64                `json:"perimeter_m"`
	Capture      *captureDTO            `json:"capture"`
	Obstructions []obstructionDTO       `json:"obstructions"`
	Accuracy     accuracyDTO            `json:"accuracy"`
	Outline      []outlinePointResponse `json:"outline"`
	Version      int64                  `json:"version"`
	Status       string                 `json:"status"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

type outlinePointResponse struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type plotSummaryResponse struct {
	Count             int     `json:"count"`
	TotalAreaM2       float64 `json:"total_area_m2"`
	TotalAreaHectares float64 `json:"total_area_hectares"`
}

type plotVersionResponse struct {
	Version      int64            `json:"version"`
	Name         string           `json:"name"`
	AreaM2       float64          `json:"area_m2"`
	AreaAres     float64          `json:"area_ares"`
	AreaHectares float64          `json:"area_hectares"`
	PerimeterM   float64          `json:"perimeter_m"`
	Points       []coordinateDTO  `json:"points"`
	Capture      *captureDTO      `json:"capture"`
	Obstructions []obstructionDTO `json:"obstructions"`
	Accuracy     accuracyDTO      `json:"accuracy"`
	CreatedAt    time.Time        `json:"created_at"`
}

type previewPlotResponse struct {
	AreaM2       *float64                  `json:"area_m2"`
	AreaAres     *float64                  `json:"area_ares"`
	AreaHectares *float64                  `json:"area_hectares"`
	PerimeterM   *float64                  `json:"perimeter_m"`
	Accuracy     *accuracyDTO              `json:"accuracy"`
	Valid        bool                      `json:"valid"`
	Errors       []respond.ProblemResponse `json:"errors"`
}

func (body updatePlotRequest) toChanges() domain.PlotChanges {
	changes := domain.PlotChanges{Name: body.Name, Capture: toCapture(body.Capture)}
	if body.Points != nil {
		points := toCoordinates(*body.Points)
		changes.Points = &points
	}

	if body.Obstructions != nil {
		obstructions := toObstructions(*body.Obstructions)
		changes.Obstructions = &obstructions
	}

	return changes
}

func (body surveyRequest) toSurvey() domain.Survey {
	return domain.Survey{Capture: toCapture(body.Capture), Obstructions: toObstructions(body.Obstructions)}
}

func toCapture(capture *captureDTO) *domain.Capture {
	if capture == nil {
		return nil
	}

	return &domain.Capture{
		Zoom:    capture.Zoom,
		Layer:   domain.MapLayer(capture.Layer),
		Pointer: domain.Pointer(capture.Pointer),
	}
}

func toObstructions(obstructions []obstructionDTO) []domain.Obstruction {
	converted := make([]domain.Obstruction, len(obstructions))
	for i, obstruction := range obstructions {
		converted[i] = domain.Obstruction{Side: obstruction.Side, Reason: domain.ObstructionReason(obstruction.Reason)}
	}

	return converted
}

func toCaptureDTO(capture *domain.Capture) *captureDTO {
	if capture == nil {
		return nil
	}

	return &captureDTO{Zoom: capture.Zoom, Layer: string(capture.Layer), Pointer: string(capture.Pointer)}
}

func toObstructionDTOs(obstructions []domain.Obstruction) []obstructionDTO {
	converted := make([]obstructionDTO, len(obstructions))
	for i, obstruction := range obstructions {
		converted[i] = obstructionDTO{Side: obstruction.Side, Reason: string(obstruction.Reason)}
	}

	return converted
}

func newAccuracyDTO(accuracy domain.Accuracy) accuracyDTO {
	return accuracyDTO{
		AreaM2Margin:       accuracy.AreaMarginM2,
		AreaHectaresMargin: accuracy.AreaMarginHectares,
		Percent:            accuracy.Percent,
		Confidence:         accuracy.Confidence,
		Contributions: contributionsDTO{
			Scale:        accuracy.Contributions.Scale,
			Obstructions: accuracy.Contributions.Obstructions,
		},
		WorstSide:    accuracy.WorstSide,
		GroundShiftM: accuracy.GroundShiftM,
		Basis:        string(accuracy.Basis),
	}
}

func toCoordinates(points []coordinateDTO) []domain.Coordinate {
	coordinates := make([]domain.Coordinate, len(points))
	for i, point := range points {
		coordinates[i] = domain.Coordinate{Latitude: point.Latitude, Longitude: point.Longitude}
	}

	return coordinates
}

func toCoordinateDTOs(coordinates []domain.Coordinate) []coordinateDTO {
	points := make([]coordinateDTO, len(coordinates))
	for i, point := range coordinates {
		points[i] = coordinateDTO{Latitude: point.Latitude, Longitude: point.Longitude}
	}

	return points
}

func newPlotVersionResponses(versions []domain.PlotVersion) []plotVersionResponse {
	responses := make([]plotVersionResponse, 0, len(versions))
	for _, version := range versions {
		responses = append(responses, plotVersionResponse{
			Version:      version.Version,
			Name:         version.Name,
			AreaM2:       version.AreaM2,
			AreaAres:     domain.SquareMetersToAres(version.AreaM2),
			AreaHectares: domain.SquareMetersToHectares(version.AreaM2),
			PerimeterM:   version.PerimeterM,
			Points:       toCoordinateDTOs(version.Points),
			Capture:      toCaptureDTO(version.Survey.Capture),
			Obstructions: toObstructionDTOs(version.Survey.Obstructions),
			Accuracy:     newAccuracyDTO(version.Accuracy),
			CreatedAt:    version.CreatedAt,
		})
	}

	return responses
}

func newPlotResponse(plot domain.Plot) plotResponse {
	points := toCoordinateDTOs(plot.Points)

	return plotResponse{
		ID:           plot.ID,
		OwnerID:      plot.OwnerID,
		Name:         plot.Name,
		Points:       points,
		AreaM2:       plot.AreaM2,
		AreaAres:     domain.SquareMetersToAres(plot.AreaM2),
		AreaHectares: domain.SquareMetersToHectares(plot.AreaM2),
		PerimeterM:   plot.PerimeterM,
		Capture:      toCaptureDTO(plot.Survey.Capture),
		Obstructions: toObstructionDTOs(plot.Survey.Obstructions),
		Accuracy:     newAccuracyDTO(plot.Accuracy),
		Outline:      newOutlineResponse(plot.Points),
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

	var accuracy *accuracyDTO
	if inspection.Accuracy != nil {
		accuracy = new(newAccuracyDTO(*inspection.Accuracy))
	}

	return previewPlotResponse{
		AreaM2:       &area.AreaM2,
		AreaAres:     new(domain.SquareMetersToAres(area.AreaM2)),
		AreaHectares: new(domain.SquareMetersToHectares(area.AreaM2)),
		PerimeterM:   &area.PerimeterM,
		Accuracy:     accuracy,
		Valid:        true,
		Errors:       problems,
	}
}

func newOutlineResponse(points []domain.Coordinate) []outlinePointResponse {
	outline := domain.Outline(points)
	response := make([]outlinePointResponse, len(outline))

	for index, point := range outline {
		response[index] = outlinePointResponse{X: point.X, Y: point.Y}
	}

	return response
}
