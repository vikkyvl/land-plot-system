package postgres

import (
	"time"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

type climateRecord struct {
	Status           string                `json:"status"`
	FirstYear        int                   `json:"first_year"`
	LastYear         int                   `json:"last_year"`
	Seasons          []seasonClimateRecord `json:"seasons"`
	LastSpringFrost  *monthDayRecord       `json:"last_spring_frost"`
	FirstAutumnFrost *monthDayRecord       `json:"first_autumn_frost"`
	GrowingDays      float64               `json:"growing_days"`
}

type seasonClimateRecord struct {
	Season          string  `json:"season"`
	MeanC           float64 `json:"mean_c"`
	MeanMinC        float64 `json:"mean_min_c"`
	MeanMaxC        float64 `json:"mean_max_c"`
	PrecipitationMm float64 `json:"precipitation_mm"`
	FrostDays       float64 `json:"frost_days"`
	HeatDays        float64 `json:"heat_days"`
}

type monthDayRecord struct {
	Month int `json:"month"`
	Day   int `json:"day"`
}

func toClimateRecord(climate domain.Climate) climateRecord {
	seasons := make([]seasonClimateRecord, len(climate.Seasons))
	for i, season := range climate.Seasons {
		seasons[i] = seasonClimateRecord{
			Season:          string(season.Season),
			MeanC:           season.MeanC,
			MeanMinC:        season.MeanMinC,
			MeanMaxC:        season.MeanMaxC,
			PrecipitationMm: season.PrecipitationMm,
			FrostDays:       season.FrostDays,
			HeatDays:        season.HeatDays,
		}
	}

	return climateRecord{
		Status:           string(climate.Status),
		FirstYear:        climate.FirstYear,
		LastYear:         climate.LastYear,
		Seasons:          seasons,
		LastSpringFrost:  toMonthDayRecord(climate.LastSpringFrost),
		FirstAutumnFrost: toMonthDayRecord(climate.FirstAutumnFrost),
		GrowingDays:      climate.GrowingDays,
	}
}

func (record climateRecord) climate() domain.Climate {
	var seasons []domain.SeasonClimate
	for _, season := range record.Seasons {
		seasons = append(seasons, domain.SeasonClimate{
			Season:          domain.Season(season.Season),
			MeanC:           season.MeanC,
			MeanMinC:        season.MeanMinC,
			MeanMaxC:        season.MeanMaxC,
			PrecipitationMm: season.PrecipitationMm,
			FrostDays:       season.FrostDays,
			HeatDays:        season.HeatDays,
		})
	}

	return domain.Climate{
		Status:           domain.DataStatus(record.Status),
		FirstYear:        record.FirstYear,
		LastYear:         record.LastYear,
		Seasons:          seasons,
		LastSpringFrost:  record.LastSpringFrost.monthDay(),
		FirstAutumnFrost: record.FirstAutumnFrost.monthDay(),
		GrowingDays:      record.GrowingDays,
	}
}

func toMonthDayRecord(monthDay *domain.MonthDay) *monthDayRecord {
	if monthDay == nil {
		return nil
	}

	return &monthDayRecord{Month: int(monthDay.Month), Day: monthDay.Day}
}

func (record *monthDayRecord) monthDay() *domain.MonthDay {
	if record == nil {
		return nil
	}

	return &domain.MonthDay{Month: time.Month(record.Month), Day: record.Day}
}
