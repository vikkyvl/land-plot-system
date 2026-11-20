package domain

import (
	"math"
	"slices"
	"time"
)

const (
	FrostThresholdC   = 0.0
	HeatThresholdC    = 30.0
	GrowingThresholdC = 5.0
	referenceYear     = 2001
	monthsInSeason    = 3
	medianDivisor     = 2
	tenthsPerDegree   = 10
)

var growingSeasonStart = time.July

type Season string

const (
	SeasonWinter Season = "winter"
	SeasonSpring Season = "spring"
	SeasonSummer Season = "summer"
	SeasonAutumn Season = "autumn"
)

var seasons = []Season{SeasonWinter, SeasonSpring, SeasonSummer, SeasonAutumn}

type DailyWeather struct {
	Date            time.Time
	MeanC           float64
	MinC            float64
	MaxC            float64
	PrecipitationMm float64
}

type MonthDay struct {
	Month time.Month
	Day   int
}

type SeasonClimate struct {
	Season          Season
	MeanC           float64
	MeanMinC        float64
	MeanMaxC        float64
	PrecipitationMm float64
	FrostDays       float64
	HeatDays        float64
}

type Climate struct {
	Status           DataStatus
	FirstYear        int
	LastYear         int
	Seasons          []SeasonClimate
	LastSpringFrost  *MonthDay
	FirstAutumnFrost *MonthDay
	GrowingDays      float64
}

type ClimateCell struct {
	LatitudeTenths  int
	LongitudeTenths int
}

type ClimatePeriod struct {
	FirstYear int
	LastYear  int
}

type seasonYear struct {
	season Season
	year   int
}

type seasonTotals struct {
	days                       int
	mean, minimum, maximum     float64
	precipitation, frost, heat float64
}

type yearTotals struct {
	lastSpringFrost  int
	firstAutumnFrost int
	growingDays      float64
}

func UnavailableClimate(period ClimatePeriod) Climate {
	return Climate{Status: DataUnavailable, FirstYear: period.FirstYear, LastYear: period.LastYear}
}

func CompletedYears(now time.Time, years int) ClimatePeriod {
	lastYear := now.Year() - 1

	return ClimatePeriod{FirstYear: lastYear - years + 1, LastYear: lastYear}
}

func (period ClimatePeriod) Start() time.Time {
	return time.Date(period.FirstYear-1, time.December, 1, 0, 0, 0, 0, time.UTC)
}

func (period ClimatePeriod) End() time.Time {
	return time.Date(period.LastYear, time.December, 31, 0, 0, 0, 0, time.UTC)
}

func CellOf(point Coordinate) ClimateCell {
	return ClimateCell{
		LatitudeTenths:  int(math.Round(point.Latitude * tenthsPerDegree)),
		LongitudeTenths: int(math.Round(point.Longitude * tenthsPerDegree)),
	}
}

func (cell ClimateCell) Centre() Coordinate {
	return Coordinate{
		Latitude:  float64(cell.LatitudeTenths) / tenthsPerDegree,
		Longitude: float64(cell.LongitudeTenths) / tenthsPerDegree,
	}
}

func ProfileClimate(days []DailyWeather, period ClimatePeriod) Climate {
	bySeason := make(map[seasonYear]*seasonTotals)
	byYear := make(map[int]*yearTotals)

	for _, day := range days {
		addSeasonDay(bySeason, day, period)
		addYearDay(byYear, day, period)
	}

	if len(bySeason) == 0 {
		return UnavailableClimate(period)
	}

	return Climate{
		Status:           DataAvailable,
		FirstYear:        period.FirstYear,
		LastYear:         period.LastYear,
		Seasons:          seasonAverages(bySeason),
		LastSpringFrost:  medianDate(byYear, func(totals *yearTotals) int { return totals.lastSpringFrost }),
		FirstAutumnFrost: medianDate(byYear, func(totals *yearTotals) int { return totals.firstAutumnFrost }),
		GrowingDays:      averageGrowingDays(byYear),
	}
}

func addSeasonDay(bySeason map[seasonYear]*seasonTotals, day DailyWeather, period ClimatePeriod) {
	key := seasonOf(day.Date)
	if key.year < period.FirstYear || key.year > period.LastYear {
		return
	}

	totals := bySeason[key]
	if totals == nil {
		totals = &seasonTotals{}
		bySeason[key] = totals
	}

	totals.days++
	totals.mean += day.MeanC
	totals.minimum += day.MinC
	totals.maximum += day.MaxC
	totals.precipitation += day.PrecipitationMm
	totals.frost += indicator(day.MinC < FrostThresholdC)
	totals.heat += indicator(day.MaxC >= HeatThresholdC)
}

func addYearDay(byYear map[int]*yearTotals, day DailyWeather, period ClimatePeriod) {
	year := day.Date.Year()
	if year < period.FirstYear || year > period.LastYear {
		return
	}

	totals := byYear[year]
	if totals == nil {
		totals = &yearTotals{}
		byYear[year] = totals
	}

	totals.growingDays += indicator(day.MeanC >= GrowingThresholdC)

	if day.MinC >= FrostThresholdC {
		return
	}

	ordinal := referenceOrdinal(day.Date.Month(), day.Date.Day())
	if day.Date.Month() < growingSeasonStart {
		totals.lastSpringFrost = max(totals.lastSpringFrost, ordinal)
	} else if totals.firstAutumnFrost == 0 || ordinal < totals.firstAutumnFrost {
		totals.firstAutumnFrost = ordinal
	}
}

func seasonOf(date time.Time) seasonYear {
	month := date.Month()
	if month == time.December {
		return seasonYear{season: SeasonWinter, year: date.Year() + 1}
	}

	return seasonYear{season: seasons[int(month)/monthsInSeason], year: date.Year()}
}

func seasonAverages(bySeason map[seasonYear]*seasonTotals) []SeasonClimate {
	averages := make([]SeasonClimate, 0, len(seasons))

	for _, season := range seasons {
		average := SeasonClimate{Season: season}
		years := 0.0

		for key, totals := range bySeason {
			if key.season != season {
				continue
			}

			days := float64(totals.days)
			average.MeanC += totals.mean / days
			average.MeanMinC += totals.minimum / days
			average.MeanMaxC += totals.maximum / days
			average.PrecipitationMm += totals.precipitation
			average.FrostDays += totals.frost
			average.HeatDays += totals.heat
			years++
		}

		if years > 0 {
			average.MeanC /= years
			average.MeanMinC /= years
			average.MeanMaxC /= years
			average.PrecipitationMm /= years
			average.FrostDays /= years
			average.HeatDays /= years
		}

		averages = append(averages, average)
	}

	return averages
}

func medianDate(byYear map[int]*yearTotals, ordinalOf func(*yearTotals) int) *MonthDay {
	var ordinals []int

	for _, totals := range byYear {
		if ordinal := ordinalOf(totals); ordinal > 0 {
			ordinals = append(ordinals, ordinal)
		}
	}

	if len(ordinals) == 0 {
		return nil
	}

	slices.Sort(ordinals)
	middle := len(ordinals) / medianDivisor
	ordinal := ordinals[middle]

	if len(ordinals)%medianDivisor == 0 {
		ordinal = (ordinals[middle-1] + ordinals[middle]) / medianDivisor
	}

	date := time.Date(referenceYear, time.January, ordinal, 0, 0, 0, 0, time.UTC)

	return &MonthDay{Month: date.Month(), Day: date.Day()}
}

func averageGrowingDays(byYear map[int]*yearTotals) float64 {
	if len(byYear) == 0 {
		return 0
	}

	var total float64
	for _, totals := range byYear {
		total += totals.growingDays
	}

	return total / float64(len(byYear))
}

func referenceOrdinal(month time.Month, day int) int {
	return time.Date(referenceYear, month, day, 0, 0, 0, 0, time.UTC).YearDay()
}

func indicator(condition bool) float64 {
	if condition {
		return 1
	}

	return 0
}
