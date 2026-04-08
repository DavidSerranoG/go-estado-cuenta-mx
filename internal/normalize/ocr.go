package normalize

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	ocrPageMarkerPattern = regexp.MustCompile(`(?i)^p[áa]gina\s*\d+\s*(?:de)?\s*\d+$`)
)

// NormalizeExtractedText applies conservative OCR-oriented cleanup while
// preserving the overall line structure expected by bank parsers.
func NormalizeExtractedText(value string) string {
	if value == "" {
		return ""
	}

	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")

	rawLines := strings.Split(value, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, raw := range rawLines {
		line := NormalizeOCRLine(raw)
		if line == "" {
			continue
		}
		if isRepeatedPageMarker(line) {
			continue
		}
		lines = append(lines, line)
	}

	lines = mergeSplitAmountLines(lines)
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// NormalizeOCRLine removes common OCR noise while keeping the semantic content
// of a single extracted line intact.
func NormalizeOCRLine(value string) string {
	if value == "" {
		return ""
	}

	replacer := strings.NewReplacer(
		"\u00a0", " ",
		"\u2007", " ",
		"\u202f", " ",
		"_", " ",
		"[", " ",
		"]", " ",
		"|", " ",
		"“", " ",
		"”", " ",
		"‘", "'",
		"’", "'",
		"—", " ",
		"–", " ",
		"`", " ",
	)

	clean := replacer.Replace(value)
	clean = strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\r':
			return -1
		case unicode.IsControl(r) && r != '\t':
			return ' '
		default:
			return r
		}
	}, clean)

	return CollapseWhitespace(clean)
}

// NormalizeOCRAmountLine repairs common OCR artifacts between a sign and the
// amount token, such as `+1$453.85` or `+|$822.45`.
func NormalizeOCRAmountLine(line string) string {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 {
		return trimmed
	}

	sign := trimmed[0]
	if sign != '+' && sign != '-' {
		return trimmed
	}

	dollar := strings.IndexByte(trimmed, '$')
	if dollar <= 1 {
		return trimmed
	}

	between := strings.TrimSpace(trimmed[1:dollar])
	if between == "" || !isOCRAmountNoise(between) {
		return trimmed
	}

	return string(sign) + trimmed[dollar:]
}

// NormalizeOCRMoneyToken repairs common OCR separators in money values so they
// can be parsed by ParseMoneyToCents.
func NormalizeOCRMoneyToken(value string) string {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return ""
	}

	clean = strings.ReplaceAll(clean, "$", "")
	clean = strings.ReplaceAll(clean, " ", "")
	clean = strings.TrimLeft(clean, "|)")

	if strings.Count(clean, ",") == 1 && !strings.Contains(clean, ".") {
		parts := strings.Split(clean, ",")
		if len(parts) == 2 && len(parts[1]) == 2 && allDigits(parts[0]) && allDigits(parts[1]) {
			clean = parts[0] + "." + parts[1]
		}
	}

	if strings.Count(clean, ".") > 1 {
		parts := strings.Split(clean, ".")
		last := parts[len(parts)-1]
		prefix := strings.Join(parts[:len(parts)-1], "")
		if len(last) == 2 && allDigits(prefix) && allDigits(last) {
			clean = prefix + "." + last
		}
	}

	if strings.Count(clean, ",") > 1 && !strings.Contains(clean, ".") {
		parts := strings.Split(clean, ",")
		last := parts[len(parts)-1]
		prefix := strings.Join(parts[:len(parts)-1], "")
		if len(last) == 2 && allDigits(prefix) && allDigits(last) {
			clean = prefix + "." + last
		}
	}

	if !strings.Contains(clean, ".") {
		if allDigits(clean) && len(clean) > 2 {
			clean = clean[:len(clean)-2] + "." + clean[len(clean)-2:]
		}
	}

	if !strings.Contains(clean, ".") {
		for i := len(clean) - 3; i > 0; i-- {
			if clean[i] != ',' {
				continue
			}

			prefix := strings.ReplaceAll(clean[:i], ",", "")
			suffix := clean[i+1:]
			if len(suffix) == 2 && allDigits(prefix) && allDigits(suffix) {
				clean = prefix + "." + suffix
				break
			}
		}
	}

	return clean
}

// ParseOCRMoneyToCents applies OCR-aware repairs and converts the result into
// integer cents.
func ParseOCRMoneyToCents(value string) (int64, error) {
	clean := NormalizeOCRMoneyToken(value)
	if clean == "" {
		return 0, fmt.Errorf("empty amount")
	}

	return ParseMoneyToCents(clean)
}

func mergeSplitAmountLines(lines []string) []string {
	if len(lines) < 2 {
		return lines
	}

	merged := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if (line == "+" || line == "-") && i+1 < len(lines) {
			next := NormalizeOCRMoneyToken(lines[i+1])
			if looksLikeOCRAmount(next) {
				merged = append(merged, line+"$"+next)
				i++
				continue
			}
		}
		merged = append(merged, line)
	}

	return merged
}

func isRepeatedPageMarker(line string) bool {
	folded := strings.ToLower(line)
	folded = strings.ReplaceAll(folded, ".", "")
	folded = strings.ReplaceAll(folded, "á", "a")
	folded = strings.ReplaceAll(folded, " ", "")
	folded = strings.ReplaceAll(folded, "de", " de ")
	folded = CollapseWhitespace(folded)
	return ocrPageMarkerPattern.MatchString(folded)
}

func looksLikeOCRAmount(value string) bool {
	if value == "" {
		return false
	}

	digits := 0
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case r == '.' || r == ',':
		default:
			return false
		}
	}

	return digits >= 3
}

func isOCRAmountNoise(value string) bool {
	for _, r := range value {
		switch r {
		case '|', '[', ']', 'I', 'i', 'l', 'L', 'T', 't', '1', '4', ' ', ')':
			continue
		default:
			return false
		}
	}

	return value != ""
}

func allDigits(value string) bool {
	if value == "" {
		return false
	}

	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}
