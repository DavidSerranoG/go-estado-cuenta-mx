package fixturegen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Overrides struct {
	Regions          []MaskRegion         `json:"regions"`
	LineReplacements []LineReplacement    `json:"line_replacements"`
	Files            map[string]Overrides `json:"files"`
}

type MaskRegion struct {
	Page  int     `json:"page"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	W     float64 `json:"w"`
	H     float64 `json:"h"`
	Label string  `json:"label,omitempty"`
}

type LineReplacement struct {
	Match   string `json:"match"`
	Replace string `json:"replace"`
}

func loadOverrides(root, bank, layout, file string) (Overrides, error) {
	if root == "" || bank == "" || layout == "" {
		return Overrides{}, nil
	}

	path := filepath.Join(root, bank, layout+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Overrides{}, nil
		}
		return Overrides{}, fmt.Errorf("read overrides: %w", err)
	}

	var base Overrides
	if err := json.Unmarshal(data, &base); err != nil {
		return Overrides{}, fmt.Errorf("decode overrides: %w", err)
	}

	if file == "" || len(base.Files) == 0 {
		return base, nil
	}

	fileBase := filepath.Base(file)
	if fileOverride, ok := base.Files[fileBase]; ok {
		base.Regions = append(base.Regions, fileOverride.Regions...)
		base.LineReplacements = append(base.LineReplacements, fileOverride.LineReplacements...)
	}

	return base, nil
}
