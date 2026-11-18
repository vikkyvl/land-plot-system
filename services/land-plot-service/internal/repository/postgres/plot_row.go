package postgres

import (
	"encoding/json"
	"fmt"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

const plotColumns = "id, owner_id, name, points, area_m2, perimeter_m, version, status, created_at, updated_at, " +
	"deleted_at, capture, obstructions"

const plotVersionColumns = "plot_id, version, name, points, area_m2, perimeter_m, created_at, capture, obstructions"

type pointRecord struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
}

type captureRecord struct {
	Zoom    float64 `json:"zoom"`
	Layer   string  `json:"layer"`
	Pointer string  `json:"pointer"`
}

type obstructionRecord struct {
	Side   int    `json:"side"`
	Reason string `json:"reason"`
}

type surveyColumns struct {
	capture      []byte
	obstructions []byte
}

type rowScanner interface {
	Scan(destinations ...any) error
}

func scanPlot(row rowScanner) (domain.Plot, error) {
	var (
		plot   domain.Plot
		points []byte
		survey surveyColumns
	)

	err := row.Scan(
		&plot.ID, &plot.OwnerID, &plot.Name, &points, &plot.AreaM2,
		&plot.PerimeterM, &plot.Version, &plot.Status, &plot.CreatedAt, &plot.UpdatedAt, &plot.DeletedAt,
		&survey.capture, &survey.obstructions,
	)
	if err != nil {
		return domain.Plot{}, err
	}

	if plot.Points, err = decodePoints(points); err != nil {
		return domain.Plot{}, err
	}

	if plot.Survey, err = survey.decode(); err != nil {
		return domain.Plot{}, err
	}

	return plot, nil
}

func encodePoints(points []domain.Coordinate) ([]byte, error) {
	encoded, err := json.Marshal(toPointRecords(points))
	if err != nil {
		return nil, fmt.Errorf(wrapEncodePlotPoints, err)
	}

	return encoded, nil
}

func decodePoints(encoded []byte) ([]domain.Coordinate, error) {
	var records []pointRecord
	if err := json.Unmarshal(encoded, &records); err != nil {
		return nil, fmt.Errorf(wrapDecodePlotPoints, err)
	}

	return toCoordinates(records), nil
}

func toPointRecords(points []domain.Coordinate) []pointRecord {
	records := make([]pointRecord, len(points))
	for i, point := range points {
		records[i] = pointRecord{Latitude: point.Latitude, Longitude: point.Longitude}
	}

	return records
}

func toCoordinates(records []pointRecord) []domain.Coordinate {
	points := make([]domain.Coordinate, len(records))
	for i, record := range records {
		points[i] = domain.Coordinate{Latitude: record.Latitude, Longitude: record.Longitude}
	}

	return points
}

func scanPlotVersion(row rowScanner) (domain.PlotVersion, error) {
	var (
		version domain.PlotVersion
		points  []byte
		survey  surveyColumns
	)

	err := row.Scan(
		&version.PlotID, &version.Version, &version.Name, &points,
		&version.AreaM2, &version.PerimeterM, &version.CreatedAt, &survey.capture, &survey.obstructions,
	)
	if err != nil {
		return domain.PlotVersion{}, err
	}

	if version.Points, err = decodePoints(points); err != nil {
		return domain.PlotVersion{}, err
	}

	if version.Survey, err = survey.decode(); err != nil {
		return domain.PlotVersion{}, err
	}

	return version, nil
}

func encodeSurvey(survey domain.Survey) (surveyColumns, error) {
	var columns surveyColumns

	if survey.Capture != nil {
		capture, err := json.Marshal(toCaptureRecord(survey.Capture))
		if err != nil {
			return surveyColumns{}, fmt.Errorf(wrapEncodePlotSurvey, err)
		}

		columns.capture = capture
	}

	obstructions, err := json.Marshal(toObstructionRecords(survey.Obstructions))
	if err != nil {
		return surveyColumns{}, fmt.Errorf(wrapEncodePlotSurvey, err)
	}

	columns.obstructions = obstructions

	return columns, nil
}

func (columns surveyColumns) decode() (domain.Survey, error) {
	var (
		capture      *captureRecord
		obstructions []obstructionRecord
	)

	if columns.capture != nil {
		if err := json.Unmarshal(columns.capture, &capture); err != nil {
			return domain.Survey{}, fmt.Errorf(wrapDecodePlotSurvey, err)
		}
	}

	if err := json.Unmarshal(columns.obstructions, &obstructions); err != nil {
		return domain.Survey{}, fmt.Errorf(wrapDecodePlotSurvey, err)
	}

	return domain.Survey{Capture: toCapture(capture), Obstructions: toObstructions(obstructions)}, nil
}

func toCaptureRecord(capture *domain.Capture) *captureRecord {
	if capture == nil {
		return nil
	}

	return &captureRecord{Zoom: capture.Zoom, Layer: string(capture.Layer), Pointer: string(capture.Pointer)}
}

func toCapture(record *captureRecord) *domain.Capture {
	if record == nil {
		return nil
	}

	return &domain.Capture{
		Zoom:    record.Zoom,
		Layer:   domain.MapLayer(record.Layer),
		Pointer: domain.Pointer(record.Pointer),
	}
}

func toObstructionRecords(obstructions []domain.Obstruction) []obstructionRecord {
	records := make([]obstructionRecord, len(obstructions))
	for i, obstruction := range obstructions {
		records[i] = obstructionRecord{Side: obstruction.Side, Reason: string(obstruction.Reason)}
	}

	return records
}

func toObstructions(records []obstructionRecord) []domain.Obstruction {
	if len(records) == 0 {
		return nil
	}

	obstructions := make([]domain.Obstruction, len(records))
	for i, record := range records {
		obstructions[i] = domain.Obstruction{Side: record.Side, Reason: domain.ObstructionReason(record.Reason)}
	}

	return obstructions
}
