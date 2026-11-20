package domain

import (
	"math"
	"testing"
	"time"
)

var (
	testClimateNow    = time.Date(2026, time.November, 18, 12, 0, 0, 0, time.UTC)
	testClimatePeriod = ClimatePeriod{FirstYear: 2016, LastYear: 2025}
)

func seasonalMean(month time.Month) float64 {
	switch {
	case month == time.December || month <= time.February:
		return -5
	case month <= time.May:
		return 10
	case month <= time.August:
		return 25
	default:
		return 8
	}
}

func syntheticWeather(period ClimatePeriod) []DailyWeather {
	var days []DailyWeather

	for date := period.Start(); !date.After(period.End()); date = date.AddDate(0, 0, 1) {
		mean := seasonalMean(date.Month())
		days = append(days, DailyWeather{Date: date, MeanC: mean, MinC: mean - 5, MaxC: mean + 5, PrecipitationMm: 1})
	}

	return days
}

func TestCompletedYearsAreTheLastTenCalendarYears(t *testing.T) {
	period := CompletedYears(testClimateNow, 10)
	if period != testClimatePeriod {
		t.Fatalf("CompletedYears() = %+v", period)
	}

	if period.Start() != time.Date(2015, time.December, 1, 0, 0, 0, 0, time.UTC) ||
		period.End() != time.Date(2025, time.December, 31, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("range %v – %v", period.Start(), period.End())
	}
}

func TestProfileClimateAveragesSeasons(t *testing.T) {
	climate := ProfileClimate(syntheticWeather(testClimatePeriod), testClimatePeriod)
	if climate.Status != DataAvailable || climate.FirstYear != 2016 || climate.LastYear != 2025 ||
		len(climate.Seasons) != 4 {
		t.Fatalf("climate = %+v", climate)
	}

	want := map[Season]SeasonClimate{
		SeasonWinter: {MeanC: -5, MeanMinC: -10, MeanMaxC: 0, PrecipitationMm: 90.3, FrostDays: 90.3, HeatDays: 0},
		SeasonSpring: {MeanC: 10, MeanMinC: 5, MeanMaxC: 15, PrecipitationMm: 92, FrostDays: 0, HeatDays: 0},
		SeasonSummer: {MeanC: 25, MeanMinC: 20, MeanMaxC: 30, PrecipitationMm: 92, FrostDays: 0, HeatDays: 92},
		SeasonAutumn: {MeanC: 8, MeanMinC: 3, MeanMaxC: 13, PrecipitationMm: 91, FrostDays: 0, HeatDays: 0},
	}
	for _, season := range climate.Seasons {
		expected := want[season.Season]
		expected.Season = season.Season

		if !closeSeason(season, expected) {
			t.Errorf("%s = %+v, want %+v", season.Season, season, expected)
		}
	}
}

func TestProfileClimateFindsFrostDatesAndGrowingDays(t *testing.T) {
	climate := ProfileClimate(syntheticWeather(testClimatePeriod), testClimatePeriod)

	if climate.LastSpringFrost == nil || *climate.LastSpringFrost != (MonthDay{Month: time.February, Day: 28}) {
		t.Fatalf("last spring frost = %v", climate.LastSpringFrost)
	}

	if climate.FirstAutumnFrost == nil || *climate.FirstAutumnFrost != (MonthDay{Month: time.December, Day: 1}) {
		t.Fatalf("first autumn frost = %v", climate.FirstAutumnFrost)
	}

	if climate.GrowingDays != 275 {
		t.Fatalf("growing days = %v, want 275", climate.GrowingDays)
	}
}

func TestProfileClimateWithoutFrostOrData(t *testing.T) {
	var warm []DailyWeather

	for _, day := range syntheticWeather(testClimatePeriod) {
		day.MinC = 1
		warm = append(warm, day)
	}

	climate := ProfileClimate(warm, testClimatePeriod)
	if climate.LastSpringFrost != nil || climate.FirstAutumnFrost != nil || climate.Seasons[0].FrostDays != 0 {
		t.Fatalf("climate = %+v", climate)
	}

	if empty := ProfileClimate(nil, testClimatePeriod); empty.Status != DataUnavailable || empty.FirstYear != 2016 {
		t.Fatalf("empty climate = %+v", empty)
	}
}

func TestCellOfRoundsToTenthOfDegree(t *testing.T) {
	cell := CellOf(Coordinate{Latitude: 50.4501, Longitude: 30.5234})
	if cell != (ClimateCell{LatitudeTenths: 505, LongitudeTenths: 305}) ||
		cell.Centre() != (Coordinate{Latitude: 50.5, Longitude: 30.5}) {
		t.Fatalf("cell = %+v centre = %+v", cell, cell.Centre())
	}
}

func closeSeason(got, want SeasonClimate) bool {
	values := [][2]float64{
		{got.MeanC, want.MeanC},
		{got.MeanMinC, want.MeanMinC},
		{got.MeanMaxC, want.MeanMaxC},
		{got.PrecipitationMm, want.PrecipitationMm},
		{got.FrostDays, want.FrostDays},
		{got.HeatDays, want.HeatDays},
	}
	for _, pair := range values {
		if math.Abs(pair[0]-pair[1]) > 1e-9 {
			return false
		}
	}

	return got.Season == want.Season
}
