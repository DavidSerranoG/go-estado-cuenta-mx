package normalize

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseMoneyToCents converts a money string like 1,234.56 into cents.
func ParseMoneyToCents(value string) (int64, error) {
	clean := strings.TrimSpace(strings.ReplaceAll(value, ",", ""))
	if clean == "" {
		return 0, fmt.Errorf("empty amount")
	}

	negative := false
	if strings.HasPrefix(clean, "-") {
		negative = true
		clean = strings.TrimPrefix(clean, "-")
	}

	parts := strings.Split(clean, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid amount %q", value)
	}

	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse whole amount: %w", err)
	}

	var fraction int64
	if len(parts) == 2 {
		if len(parts[1]) != 2 {
			return 0, fmt.Errorf("invalid cents in amount %q", value)
		}
		fraction, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse fractional amount: %w", err)
		}
	}

	cents := (whole * 100) + fraction
	if negative {
		cents = -cents
	}

	return cents, nil
}
