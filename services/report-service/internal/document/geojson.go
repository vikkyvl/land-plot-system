package document

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

const (
	typeFeatureCollection = "FeatureCollection"
	typeFeature           = "Feature"
	typePolygon           = "Polygon"
)

type geoJSONCollection struct {
	Type     string           `json:"type"`
	Features []geoJSONFeature `json:"features"`
}

type geoJSONFeature struct {
	Type       string            `json:"type"`
	Geometry   geoJSONGeometry   `json:"geometry"`
	Properties geoJSONProperties `json:"properties"`
}

type geoJSONGeometry struct {
	Type        string         `json:"type"`
	Coordinates [][][2]float64 `json:"coordinates"`
}

type geoJSONProperties struct {
	ReportID     string    `json:"report_id"`
	ReportNumber string    `json:"report_number"`
	PlotID       string    `json:"plot_id"`
	PlotName     string    `json:"plot_name"`
	PlotVersion  int64     `json:"plot_version"`
	AreaM2       float64   `json:"area_m2"`
	AreaHectares float64   `json:"area_hectares"`
	PerimeterM   float64   `json:"perimeter_m"`
	CreatedAt    time.Time `json:"created_at"`
}

func (Renderer) GeoJSON(report domain.Report) ([]byte, error) {
	if len(report.Boundary) == 0 {
		return nil, domain.ErrBoundaryNotRecorded
	}

	ring := make([][2]float64, 0, len(report.Boundary)+1)
	for _, point := range report.Boundary {
		ring = append(ring, [2]float64{point.Longitude, point.Latitude})
	}

	ring = append(ring, ring[0])

	collection := geoJSONCollection{
		Type: typeFeatureCollection,
		Features: []geoJSONFeature{{
			Type:     typeFeature,
			Geometry: geoJSONGeometry{Type: typePolygon, Coordinates: [][][2]float64{ring}},
			Properties: geoJSONProperties{
				ReportID:     report.ID,
				ReportNumber: report.Number(),
				PlotID:       report.PlotID,
				PlotName:     report.PlotName,
				PlotVersion:  report.PlotVersion,
				AreaM2:       report.AreaM2,
				AreaHectares: report.AreaHectares,
				PerimeterM:   report.PerimeterM,
				CreatedAt:    report.CreatedAt,
			},
		}},
	}

	encoded, err := json.MarshalIndent(collection, "", "  ")
	if err != nil {
		return nil, fmt.Errorf(wrapEncodeGeoJSON, err)
	}

	return encoded, nil
}
