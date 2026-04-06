package normalize

import (
	"strings"
	"time"
)

// ParseDateDDMMYYYY parses dates in dd/mm/yyyy format using UTC.
func ParseDateDDMMYYYY(value string) (time.Time, error) {
	return time.Parse("02/01/2006", strings.TrimSpace(value))
}
