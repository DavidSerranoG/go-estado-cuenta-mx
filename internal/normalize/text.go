package normalize

import "strings"

// CollapseWhitespace trims the string and reduces runs of whitespace to single spaces.
func CollapseWhitespace(value string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(value), " "))
}
