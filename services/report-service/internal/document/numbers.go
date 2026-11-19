package document

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

const (
	digitGroup          = 3
	percentOfWhole      = 100
	pluralFew           = 1
	pluralMany          = 2
	tens                = 10
	hundreds            = 100
	teenStart           = 11
	teenEnd             = 14
	fewStart            = 2
	fewEnd              = 4
	coordinateDecimals  = 5
	hectareDecimals     = 2
	smallestHectareRest = 0.005
)

func (text texts) number(value float64, decimals int) string {
	formatted := strconv.FormatFloat(math.Abs(value), 'f', decimals, 64)
	whole, fraction, _ := strings.Cut(formatted, ".")

	var grouped strings.Builder

	for index, digit := range whole {
		if index > 0 && (len(whole)-index)%digitGroup == 0 {
			grouped.WriteString(text.groupSeparator)
		}

		grouped.WriteRune(digit)
	}

	result := grouped.String()
	if fraction != "" {
		result += text.decimalSeparator + fraction
	}

	if value < 0 && strings.Trim(result, "0"+text.decimalSeparator+text.groupSeparator) != "" {
		result = "−" + result
	}

	return result
}

func (text texts) formatHectares(value float64) string {
	return fmt.Sprintf(text.hectares, text.number(value, hectareDecimals))
}

func (text texts) formatMargin(accuracy domain.AreaAccuracy) string {
	value := fmt.Sprintf(text.squareMeters, text.number(accuracy.AreaMarginM2, 0))
	if accuracy.AreaMarginHectares >= smallestHectareRest {
		value = text.formatHectares(accuracy.AreaMarginHectares)
	}

	if accuracy.Basis == "assumed" {
		return fmt.Sprintf(text.approximateMargin, value)
	}

	return fmt.Sprintf(text.margin, value)
}

func (text texts) percentOf(share float64) string {
	return fmt.Sprintf(text.percent, text.number(share*percentOfWhole, 0))
}

func (text texts) percentValue(value float64) string {
	return fmt.Sprintf(text.percent, text.number(value, 1))
}

func (text texts) formatMeters(value float64) string {
	return fmt.Sprintf(text.meters, text.number(value, 0))
}

func (text texts) date(value time.Time) string {
	return value.Format(text.dateLayout)
}

func (text texts) monthDay(value *domain.MonthDay) string {
	if value == nil {
		return text.noFrost
	}

	return fmt.Sprintf("%d %s", value.Day, text.months[int(value.Month)-1])
}

func (text texts) growing(days float64) string {
	count := int(math.Round(days))

	return fmt.Sprintf(text.growingDays[text.plural(count)], text.number(float64(count), 0))
}

func (text texts) coordinate(value float64) string {
	return strconv.FormatFloat(value, 'f', coordinateDecimals, 64)
}

func ukrainianPlural(count int) int {
	lastTwo := count % hundreds
	last := count % tens

	switch {
	case lastTwo >= teenStart && lastTwo <= teenEnd:
		return pluralMany
	case last == 1:
		return 0
	case last >= fewStart && last <= fewEnd:
		return pluralFew
	default:
		return pluralMany
	}
}

func englishPlural(count int) int {
	if count == 1 {
		return 0
	}

	return pluralMany
}

func (text texts) formatCelsius(value float64) string {
	return fmt.Sprintf(text.celsius, text.number(value, 1))
}
