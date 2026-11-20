package postgres

import (
	"encoding/json"
	"fmt"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

type snapshotColumns struct {
	accuracy []byte
	terrain  []byte
	climate  []byte
	boundary []byte
}

type boundaryPointRecord struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
}

type accuracyRecord struct {
	AreaMarginM2       float64 `json:"area_m2_margin"`
	AreaMarginHectares float64 `json:"area_hectares_margin"`
	Percent            float64 `json:"percent"`
	Confidence         float64 `json:"confidence"`
	ScaleShare         float64 `json:"scale_share"`
	ObstructionShare   float64 `json:"obstruction_share"`
	WorstSide          *int    `json:"worst_side"`
	GroundShiftM       float64 `json:"ground_shift_m"`
	Basis              string  `json:"basis"`
}

type terrainRecord struct {
	Status                 string  `json:"status"`
	SlopeDegrees           float64 `json:"slope_degrees"`
	SurfaceAreaM2          float64 `json:"surface_area_m2"`
	SurfaceIncreasePercent float64 `json:"surface_increase_percent"`
	MinElevationM          float64 `json:"min_elevation_m"`
	MaxElevationM          float64 `json:"max_elevation_m"`
	ReliefM                float64 `json:"relief_m"`
	Coarse                 bool    `json:"coarse"`
}

func encodeSnapshot(report domain.Report) (snapshotColumns, error) {
	var (
		columns snapshotColumns
		err     error
	)

	if report.Accuracy != nil {
		if columns.accuracy, err = json.Marshal(accuracyRecord(*report.Accuracy)); err != nil {
			return snapshotColumns{}, fmt.Errorf(wrapEncodeReportSnapshot, err)
		}
	}

	if report.Terrain != nil {
		if columns.terrain, err = json.Marshal(terrainRecord{
			Status:                 string(report.Terrain.Status),
			SlopeDegrees:           report.Terrain.SlopeDegrees,
			SurfaceAreaM2:          report.Terrain.SurfaceAreaM2,
			SurfaceIncreasePercent: report.Terrain.SurfaceIncreasePercent,
			MinElevationM:          report.Terrain.MinElevationM,
			MaxElevationM:          report.Terrain.MaxElevationM,
			ReliefM:                report.Terrain.ReliefM,
			Coarse:                 report.Terrain.Coarse,
		}); err != nil {
			return snapshotColumns{}, fmt.Errorf(wrapEncodeReportSnapshot, err)
		}
	}

	if report.Climate != nil {
		if columns.climate, err = json.Marshal(toClimateRecord(*report.Climate)); err != nil {
			return snapshotColumns{}, fmt.Errorf(wrapEncodeReportSnapshot, err)
		}
	}

	if len(report.Boundary) > 0 {
		points := make([]boundaryPointRecord, len(report.Boundary))
		for index, point := range report.Boundary {
			points[index] = boundaryPointRecord{Latitude: point.Latitude, Longitude: point.Longitude}
		}

		if columns.boundary, err = json.Marshal(points); err != nil {
			return snapshotColumns{}, fmt.Errorf(wrapEncodeReportSnapshot, err)
		}
	}

	return columns, nil
}

func (columns snapshotColumns) decodeInto(report domain.Report) (domain.Report, error) {
	if columns.accuracy != nil {
		var record accuracyRecord
		if err := json.Unmarshal(columns.accuracy, &record); err != nil {
			return domain.Report{}, fmt.Errorf(wrapDecodeReportSnapshot, err)
		}

		report.Accuracy = new(domain.AreaAccuracy(record))
	}

	if columns.terrain != nil {
		var record terrainRecord
		if err := json.Unmarshal(columns.terrain, &record); err != nil {
			return domain.Report{}, fmt.Errorf(wrapDecodeReportSnapshot, err)
		}

		report.Terrain = &domain.Terrain{
			Status:                 domain.DataStatus(record.Status),
			SlopeDegrees:           record.SlopeDegrees,
			SurfaceAreaM2:          record.SurfaceAreaM2,
			SurfaceIncreasePercent: record.SurfaceIncreasePercent,
			MinElevationM:          record.MinElevationM,
			MaxElevationM:          record.MaxElevationM,
			ReliefM:                record.ReliefM,
			Coarse:                 record.Coarse,
		}
	}

	if columns.climate != nil {
		var record climateRecord
		if err := json.Unmarshal(columns.climate, &record); err != nil {
			return domain.Report{}, fmt.Errorf(wrapDecodeReportSnapshot, err)
		}

		report.Climate = new(record.climate())
	}

	if columns.boundary != nil {
		var points []boundaryPointRecord
		if err := json.Unmarshal(columns.boundary, &points); err != nil {
			return domain.Report{}, fmt.Errorf(wrapDecodeReportSnapshot, err)
		}

		report.Boundary = make([]domain.Coordinate, len(points))
		for index, point := range points {
			report.Boundary[index] = domain.Coordinate{Latitude: point.Latitude, Longitude: point.Longitude}
		}
	}

	return report, nil
}
