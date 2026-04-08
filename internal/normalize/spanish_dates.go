package normalize

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseDateDDMonYYYYSpanish parses dates like 15-Sep-2025 using UTC.
func ParseDateDDMonYYYYSpanish(value string) (time.Time, error) {
	parts := strings.Split(strings.TrimSpace(value), "-")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid date %q", value)
	}

	day, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return time.Time{}, err
	}

	month, ok := ParseSpanishMonth(parts[1])
	if !ok {
		return time.Time{}, fmt.Errorf("invalid month %q", parts[1])
	}

	year, err := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil {
		return time.Time{}, err
	}

	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), nil
}

// ParseSpanishMonth parses common Spanish month abbreviations.
func ParseSpanishMonth(value string) (time.Month, bool) {
	switch foldSpanishToken(value) {
	case "ENE", "JAN":
		return time.January, true
	case "FEB":
		return time.February, true
	case "MAR":
		return time.March, true
	case "ABR", "APR":
		return time.April, true
	case "MAY":
		return time.May, true
	case "JUN":
		return time.June, true
	case "JUL":
		return time.July, true
	case "AGO", "AUG":
		return time.August, true
	case "SEP", "SEPT":
		return time.September, true
	case "OCT":
		return time.October, true
	case "NOV":
		return time.November, true
	case "DIC", "DEC":
		return time.December, true
	default:
		return time.Month(0), false
	}
}

func foldSpanishToken(value string) string {
	replacer := strings.NewReplacer(
		"á", "a",
		"é", "e",
		"í", "i",
		"ó", "o",
		"ú", "u",
		"Á", "A",
		"É", "E",
		"Í", "I",
		"Ó", "O",
		"Ú", "U",
		"0", "O",
		"1", "I",
		"5", "S",
		"8", "B",
		".", "",
	)

	return strings.ToUpper(strings.TrimSpace(replacer.Replace(value)))
}
