package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/DavidSerranoG/go-estado-cuenta-mx/internal/fixturegen"
)

func main() {
	var cfg fixturegen.Config

	flag.StringVar(&cfg.Input, "input", ".tmp/real-pdfs", "input pdf file or root directory")
	flag.StringVar(&cfg.Output, "output", "testdata", "output root directory")
	flag.StringVar(&cfg.Bank, "bank", "", "optional bank filter")
	flag.StringVar(&cfg.Layout, "layout", "", "optional layout filter")
	flag.StringVar(&cfg.File, "file", "", "optional single file basename filter")
	flag.Func("branding", "branding mode: mixed, neutral, preserve", func(value string) error {
		cfg.Branding = fixturegen.BrandingMode(strings.ToLower(strings.TrimSpace(value)))
		return nil
	})
	flag.Func("mode", "output mode: public, local, both", func(value string) error {
		cfg.Mode = fixturegen.OutputMode(strings.ToLower(strings.TrimSpace(value)))
		return nil
	})
	flag.BoolVar(&cfg.Check, "check", false, "validate generation without writing files")
	flag.BoolVar(&cfg.AllowLowFidelity, "allow-low-fidelity", false, "allow parser-equivalent fallback for public fixtures")
	flag.StringVar(&cfg.ReportPath, "report", "", "optional report output path")
	flag.StringVar(&cfg.ReportFormat, "report-format", "json", "report format: json or markdown")
	flag.Parse()

	if cfg.Branding == "" {
		cfg.Branding = fixturegen.BrandingMixed
	}
	if cfg.Mode == "" {
		cfg.Mode = fixturegen.OutputBoth
	}

	report, err := fixturegen.New().Generate(context.Background(), cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
