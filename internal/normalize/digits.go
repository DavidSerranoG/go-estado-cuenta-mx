package normalize

import "strings"

// DigitsOnly keeps only ASCII digits from the provided value.
func DigitsOnly(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))

	for _, r := range value {
		if r >= '0' && r <= '9' {
			builder.WriteRune(r)
		}
	}

	return builder.String()
}
