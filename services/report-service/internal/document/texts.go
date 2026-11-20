package document

import "github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"

type texts struct {
	title             string
	meta              string
	plotDeleted       string
	noData            string
	boundaryTitle     string
	areaTitle         string
	hectares          string
	squareMeters      string
	meters            string
	areaDetails       string
	margin            string
	approximateMargin string
	approximateNote   string
	accuracyDetails   string
	noAccuracy        string
	terrainTitle      string
	terrainDetails    string
	coarse            string
	cadastral         string
	climateTitle      string
	climateHeading    string
	celsius           string
	millimeters       string
	seasonColumns     [7]string
	seasons           map[domain.Season]string
	lastSpringFrost   string
	firstAutumnFrost  string
	noFrost           string
	growingDays       [3]string
	coordinatesTitle  string
	coordinateColumns [3]string
	sources           string
	disclaimer        string
	months            [12]string
	decimalSeparator  string
	groupSeparator    string
	percent           string
	dateLayout        string
	plural            func(int) int
}

var catalog = map[domain.Language]texts{
	domain.LanguageUkrainian: {
		title:             "Звіт · %s",
		meta:              "Звіт № %s · сформовано %s",
		plotDeleted:       "Ділянку видалено. Звіт збережено з даними на момент формування.",
		noData:            "Даних немає",
		boundaryTitle:     "Межа ділянки",
		areaTitle:         "Площа",
		hectares:          "%s га",
		squareMeters:      "%s м²",
		meters:            "%s м",
		areaDetails:       "у м²: %s · ари: %s · периметр: %s",
		margin:            "± %s",
		approximateMargin: "≈ ± %s",
		approximateNote:   "Орієнтовно: ділянку збережено без даних про масштаб.",
		accuracyDetails:   "Імовірність %s · масштаб карти %s · перешкоди на межі %s · можливий зсув на місцевості ± %s",
		noAccuracy:        "Похибка: даних немає",
		terrainTitle:      "Рельєф",
		terrainDetails:    "Середній ухил %s° · площа поверхні %s (+%s) · висоти %s–%s (перепад %s)",
		coarse:            "Оцінка груба: ділянка менша за 1 га, а крок моделі висот — 90 м.",
		cadastral:         "Кадастрова площа — горизонтальна проєкція, площа поверхні довідкова.",
		climateTitle:      "Клімат · середнє за %d–%d",
		climateHeading:    "Клімат",
		celsius:           "%s °C",
		millimeters:       "%s мм",
		seasonColumns:     [7]string{"Сезон", "Середня", "Мін.", "Макс.", "Опади", "Мороз, дн.", "Спека, дн."},
		seasons: map[domain.Season]string{
			domain.SeasonWinter: "Зима", domain.SeasonSpring: "Весна",
			domain.SeasonSummer: "Літо", domain.SeasonAutumn: "Осінь",
		},
		lastSpringFrost:  "Останній весняний заморозок: %s",
		firstAutumnFrost: "Перший осінній заморозок: %s",
		noFrost:          "не буває",
		growingDays: [3]string{
			"%s день із середньою температурою ≥ 5 °C",
			"%s дні із середньою температурою ≥ 5 °C",
			"%s днів із середньою температурою ≥ 5 °C",
		},
		coordinatesTitle:  "Координати точок межі (WGS 84)",
		coordinateColumns: [3]string{"№", "Широта", "Довгота"},
		sources:           "Дані: Open-Meteo.com (CC BY 4.0), Copernicus C3S ERA5, Copernicus DEM GLO-90",
		disclaimer:        "Розрахунок наближений і не є кадастровим вимірюванням.",
		months: [12]string{
			"січня", "лютого", "березня", "квітня", "травня", "червня",
			"липня", "серпня", "вересня", "жовтня", "листопада", "грудня",
		},
		decimalSeparator: ",",
		groupSeparator:   " ",
		percent:          "%s %%",
		dateLayout:       "02.01.2006",
		plural:           ukrainianPlural,
	},
	domain.LanguageEnglish: {
		title:             "Report · %s",
		meta:              "Report No. %s · created %s",
		plotDeleted:       "The plot was deleted. The report keeps the data from the time it was created.",
		noData:            "No data",
		boundaryTitle:     "Plot boundary",
		areaTitle:         "Area",
		hectares:          "%s ha",
		squareMeters:      "%s m²",
		meters:            "%s m",
		areaDetails:       "m²: %s · ares: %s · perimeter: %s",
		margin:            "± %s",
		approximateMargin: "≈ ± %s",
		approximateNote:   "Approximate: the plot was saved without map scale data.",
		accuracyDetails:   "Probability %s · map scale %s · obstructed boundary %s · possible shift on the ground ± %s",
		noAccuracy:        "Margin: no data",
		terrainTitle:      "Terrain",
		terrainDetails:    "Mean slope %s° · surface area %s (+%s) · elevations %s–%s (relief %s)",
		coarse:            "Rough estimate: the plot is smaller than 1 ha and the elevation model step is 90 m.",
		cadastral:         "The cadastral area is the horizontal projection; the surface area is for reference only.",
		climateTitle:      "Climate · average for %d–%d",
		climateHeading:    "Climate",
		celsius:           "%s °C",
		millimeters:       "%s mm",
		seasonColumns:     [7]string{"Season", "Mean", "Min", "Max", "Precipitation", "Frost days", "Hot days"},
		seasons: map[domain.Season]string{
			domain.SeasonWinter: "Winter", domain.SeasonSpring: "Spring",
			domain.SeasonSummer: "Summer", domain.SeasonAutumn: "Autumn",
		},
		lastSpringFrost:  "Last spring frost: %s",
		firstAutumnFrost: "First autumn frost: %s",
		noFrost:          "none",
		growingDays: [3]string{
			"%s day with a mean temperature ≥ 5 °C",
			"%s days with a mean temperature ≥ 5 °C",
			"%s days with a mean temperature ≥ 5 °C",
		},
		coordinatesTitle:  "Boundary coordinates (WGS 84)",
		coordinateColumns: [3]string{"No.", "Latitude", "Longitude"},
		sources:           "Data: Open-Meteo.com (CC BY 4.0), Copernicus C3S ERA5, Copernicus DEM GLO-90",
		disclaimer:        "The calculation is approximate and is not a cadastral measurement.",
		months: [12]string{
			"January", "February", "March", "April", "May", "June",
			"July", "August", "September", "October", "November", "December",
		},
		decimalSeparator: ".",
		groupSeparator:   ",",
		percent:          "%s%%",
		dateLayout:       "02/01/2006",
		plural:           englishPlural,
	},
}

func textsFor(language domain.Language) texts {
	if found, ok := catalog[language]; ok {
		return found
	}

	return catalog[domain.LanguageUkrainian]
}
