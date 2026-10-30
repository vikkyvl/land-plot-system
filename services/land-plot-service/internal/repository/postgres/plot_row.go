package postgres

import (
	"encoding/json"
	"fmt"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

const plotColumns = "id, owner_id, name, points, area_m2, perimeter_m, version, created_at, updated_at"

type pointRecord struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
}

type rowScanner interface {
	Scan(destinations ...any) error
}

func scanPlot(row rowScanner) (domain.Plot, error) {
	var (
		plot   domain.Plot
		points []byte
	)

	err := row.Scan(
		&plot.ID, &plot.OwnerID, &plot.Name, &points, &plot.AreaM2,
		&plot.PerimeterM, &plot.Version, &plot.CreatedAt, &plot.UpdatedAt,
	)
	if err != nil {
		return domain.Plot{}, err
	}

	if plot.Points, err = decodePoints(points); err != nil {
		return domain.Plot{}, err
	}

	return plot, nil
}

func encodePoints(points []domain.Coordinate) ([]byte, error) {
	records := make([]pointRecord, len(points))
	for i, point := range points {
		records[i] = pointRecord{Latitude: point.Latitude, Longitude: point.Longitude}
	}

	encoded, err := json.Marshal(records)
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

	points := make([]domain.Coordinate, len(records))
	for i, record := range records {
		points[i] = domain.Coordinate{Latitude: record.Latitude, Longitude: record.Longitude}
	}

	return points, nil
}
