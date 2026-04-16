package edocuenta_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepoFixturesDoNotContainKnownSensitiveTokens(t *testing.T) {
	t.Parallel()

	files := []string{
		"bbva/parser_test.go",
		"internal/banks/bbva/parser_internal_test.go",
		"processor_test.go",
		"README.md",
		"docs/banks/bbva.md",
	}

	banned := []string{
		strings.Join([]string{"DAVID", "ALBERTO", "SERRANO", "GARCIA"}, " "),
		"DAVID ALBERTO" + "," + "SERRANO/GARCIA",
		"SEGD" + "940531AH1",
		"C CUARTA" + " 66",
		"15289" + "07610",
		"04849" + "84080",
		"PEREZ HERNANDEZ" + " ANA LIA",
	}

	for _, rel := range files {
		data, err := os.ReadFile(filepath.Clean(rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := string(data)
		for _, token := range banned {
			if strings.Contains(text, token) {
				t.Fatalf("%s still contains banned token %q", rel, token)
			}
		}
	}
}
