package handler

import (
	"fmt"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

const monthDayFormat = "%02d-%02d"

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

type terrainResponse struct {
	Status                 string  `json:"status"`
	SlopeDegrees           float64 `json:"slope_degrees"`
	SurfaceAreaM2          float64 `json:"surface_area_m2"`
	SurfaceAreaHectares    float64 `json:"surface_area_hectares"`
	SurfaceIncreasePercent float64 `json:"surface_increase_percent"`
	MinElevationM          float64 `json:"min_elevation_m"`
	MaxElevationM          float64 `json:"max_elevation_m"`
	ReliefM                float64 `json:"relief_m"`
	Coarse                 bool    `json:"coarse"`
	Source                 string  `json:"source"`
}

type climateResponse struct {
	Status           string           `json:"status"`
	FirstYear        int              `json:"first_year"`
	LastYear         int              `json:"last_year"`
	Seasons          []seasonResponse `json:"seasons"`
	LastSpringFrost  *string          `json:"last_spring_frost"`
	FirstAutumnFrost *string          `json:"first_autumn_frost"`
	GrowingDays      float64          `json:"growing_days"`
	Source           string           `json:"source"`
}

type seasonResponse struct {
	Season          string  `json:"season"`
	MeanC           float64 `json:"mean_c"`
	MeanMinC        float64 `json:"mean_min_c"`
	MeanMaxC        float64 `json:"mean_max_c"`
	PrecipitationMm float64 `json:"precipitation_mm"`
	FrostDays       float64 `json:"frost_days"`
	HeatDays        float64 `json:"heat_days"`
}

func newAccuracyResponse(accuracy *domain.AreaAccuracy) *accuracyResponse {
	if accuracy == nil {
		return nil
	}

	return &accuracyResponse{
		AreaM2Margin:       accuracy.AreaMarginM2,
		AreaHectaresMargin: accuracy.AreaMarginHectares,
		Percent:            accuracy.Percent,
		Confidence:         accuracy.Confidence,
		Contributions:      contributionsResponse{Scale: accuracy.ScaleShare, Obstructions: accuracy.ObstructionShare},
		WorstSide:          accuracy.WorstSide,
		GroundShiftM:       accuracy.GroundShiftM,
		Basis:              accuracy.Basis,
	}
}

func newTerrainResponse(terrain *domain.Terrain) *terrainResponse {
	if terrain == nil {
		return nil
	}

	return &terrainResponse{
		Status:                 string(terrain.Status),
		SlopeDegrees:           terrain.SlopeDegrees,
		SurfaceAreaM2:          terrain.SurfaceAreaM2,
		SurfaceAreaHectares:    terrain.SurfaceAreaM2 / domain.SquareMetersPerHectare,
		SurfaceIncreasePercent: terrain.SurfaceIncreasePercent,
		MinElevationM:          terrain.MinElevationM,
		MaxElevationM:          terrain.MaxElevationM,
		ReliefM:                terrain.ReliefM,
		Coarse:                 terrain.Coarse,
		Source:                 domain.TerrainSource,
	}
}

func newClimateResponse(climate *domain.Climate) *climateResponse {
	if climate == nil {
		return nil
	}

	seasons := make([]seasonResponse, len(climate.Seasons))
	for i, season := range climate.Seasons {
		seasons[i] = seasonResponse{
			Season:          string(season.Season),
			MeanC:           season.MeanC,
			MeanMinC:        season.MeanMinC,
			MeanMaxC:        season.MeanMaxC,
			PrecipitationMm: season.PrecipitationMm,
			FrostDays:       season.FrostDays,
			HeatDays:        season.HeatDays,
		}
	}

	return &climateResponse{
		Status:           string(climate.Status),
		FirstYear:        climate.FirstYear,
		LastYear:         climate.LastYear,
		Seasons:          seasons,
		LastSpringFrost:  formatMonthDay(climate.LastSpringFrost),
		FirstAutumnFrost: formatMonthDay(climate.FirstAutumnFrost),
		GrowingDays:      climate.GrowingDays,
		Source:           domain.ClimateSource,
	}
}

func formatMonthDay(monthDay *domain.MonthDay) *string {
	if monthDay == nil {
		return nil
	}

	return new(fmt.Sprintf(monthDayFormat, int(monthDay.Month), monthDay.Day))
}
