package fixturegen

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	edocuenta "github.com/DavidSerranoG/go-estado-cuenta-mx"
	"github.com/DavidSerranoG/go-estado-cuenta-mx/supported"
)

func New() *Generator {
	return &Generator{
		processor:     supported.New(),
		bboxExtractor: NewPdftotextBBoxExtractor(),
		rasterizer:    NewOptionalRasterizer(),
		overridesRoot: filepath.Join("testdata", "fixturegen", "overrides"),
	}
}

func (g *Generator) Generate(ctx context.Context, cfg Config) (AggregateReport, error) {
	files, root, err := collectInputFiles(cfg)
	if err != nil {
		return AggregateReport{}, err
	}

	report := AggregateReport{
		Root:     root,
		Output:   cfg.Output,
		Branding: string(cfg.Branding),
		Mode:     string(cfg.Mode),
		Files:    make([]Metadata, 0, len(files)),
	}

	for _, file := range files {
		meta, err := g.GenerateFile(ctx, file, cfg)
		if err != nil {
			return report, err
		}
		report.Files = append(report.Files, meta)
	}

	if cfg.ReportPath != "" && !cfg.Check {
		if err := writeAggregateReport(cfg.ReportPath, cfg.ReportFormat, report); err != nil {
			return report, err
		}
	}

	return report, nil
}

func (g *Generator) GenerateFile(ctx context.Context, inputPath string, cfg Config) (Metadata, error) {
	pdfBytes, err := os.ReadFile(inputPath)
	if err != nil {
		return Metadata{}, fmt.Errorf("read input pdf: %w", err)
	}

	hint := deriveHint(cfg, inputPath)
	original, err := g.processor.ParsePDFResult(ctx, pdfBytes)
	if err != nil {
		return Metadata{}, fmt.Errorf("parse original pdf: %w", err)
	}

	hint = normalizeHint(hint, original)
	overrides, err := loadOverrides(g.overridesRoot, hint.Bank, hint.Layout, hint.File)
	if err != nil {
		return Metadata{}, err
	}

	ctxSan := newSanitizeContext(hint, original, overrides)

	textDoc, textErr := g.bboxExtractor.Extract(ctx, pdfBytes)
	rasterDoc, rasterErr := g.rasterizer.Rasterize(ctx, pdfBytes)

	var (
		dummyBytes []byte
		fidelity   Fidelity
		warnings   []string
	)

	if textErr == nil && rasterErr == nil {
		sanitizedText := ctxSan.sanitizeTextDocument(textDoc, cfg.Branding)
		dummyBytes, err = renderHighFidelity(sanitizedText, rasterDoc, overrides)
		if err != nil {
			return Metadata{}, err
		}
		fidelity = FidelityHigh
	} else {
		if cfg.Mode == OutputPublic && !cfg.AllowLowFidelity {
			return Metadata{}, fmt.Errorf("public mode requires high-fidelity generation: bbox=%v raster=%v", textErr, rasterErr)
		}

		if textErr != nil {
			return Metadata{}, fmt.Errorf("fallback generation requires bbox text: %w", textErr)
		}

		sanitizedText := ctxSan.sanitizeTextDocument(textDoc, cfg.Branding)
		dummyBytes, err = renderFallback(sanitizedText)
		if err != nil {
			return Metadata{}, err
		}
		fidelity = FidelityParser
		if rasterErr != nil {
			warnings = append(warnings, rasterErr.Error())
		}
	}

	dummyResult, validation := validateDummy(ctx, g.processor, hint, original, dummyBytes)
	warnings = append(warnings, dummyResult.Warnings...)
	if !validation.ParseOK {
		return Metadata{}, fmt.Errorf("dummy validation failed: %s", validation.Error)
	}

	replacements := replacementList(ctxSan.replacements)
	if leaks := findLeaks(dummyResult.ExtractedText, replacements, ctxSan); len(leaks) > 0 {
		return Metadata{}, fmt.Errorf("dummy text leaked original sensitive tokens: %s", strings.Join(leaks, ", "))
	}

	name := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	metadata := Metadata{
		InputPath:         inputPath,
		Bank:              hint.Bank,
		Layout:            hint.Layout,
		Branding:          cfg.Branding,
		Mode:              cfg.Mode,
		Fidelity:          fidelity,
		SelectedExtractor: original.Extraction.SelectedExtractor,
		ReplacementCounts: replacementCounts(ctxSan.replacements),
		Replacements:      replacements,
		Warnings:          warnings,
		Validation:        validation,
		Extra: map[string]interface{}{
			"original_transactions": len(original.Statement.Transactions),
			"dummy_transactions":    len(dummyResult.Statement.Transactions),
		},
	}
	if textErr == nil {
		metadata.BBoxTool = textDoc.Tool
	}
	if rasterErr == nil {
		metadata.Rasterizer = rasterDoc.Tool
	}

	outputDirs := resolveOutputDirs(cfg, hint)
	if len(outputDirs) == 0 {
		return Metadata{}, fmt.Errorf("no output directories resolved")
	}

	written := make([]Metadata, 0, len(outputDirs))
	for _, outputDir := range outputDirs {
		item, err := writeOutputs(outputDir, name, dummyBytes, metadata, cfg.Check)
		if err != nil {
			return Metadata{}, err
		}
		written = append(written, item)
	}

	metadata = written[0]
	if len(written) > 1 {
		outputPaths := make([]string, 0, len(written))
		sidecarPaths := make([]string, 0, len(written))
		for _, item := range written {
			outputPaths = append(outputPaths, item.OutputPath)
			sidecarPaths = append(sidecarPaths, item.SidecarPath)
		}
		metadata.Extra["all_output_paths"] = outputPaths
		metadata.Extra["all_sidecar_paths"] = sidecarPaths
	}

	return metadata, nil
}

func collectInputFiles(cfg Config) ([]string, string, error) {
	root := cfg.Input
	if root == "" {
		root = filepath.Join(".tmp", "real-pdfs")
	}

	info, err := os.Stat(root)
	if err != nil {
		return nil, root, err
	}
	if !info.IsDir() {
		return []string{root}, filepath.Dir(root), nil
	}

	var files []string
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(path), ".pdf") {
			return nil
		}

		hint := deriveHint(cfg, path)
		if cfg.Bank != "" && !strings.EqualFold(cfg.Bank, hint.Bank) {
			return nil
		}
		if cfg.Layout != "" && !strings.EqualFold(cfg.Layout, hint.Layout) {
			return nil
		}
		if cfg.File != "" && !strings.EqualFold(cfg.File, filepath.Base(path)) {
			return nil
		}

		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, root, err
	}

	sort.Strings(files)
	return files, root, nil
}

func deriveHint(cfg Config, path string) Hint {
	hint := Hint{
		Bank:   strings.ToLower(strings.TrimSpace(cfg.Bank)),
		Layout: strings.ToLower(strings.TrimSpace(cfg.Layout)),
		File:   filepath.Base(path),
	}

	parts := strings.Split(filepath.ToSlash(path), "/")
	for i := range parts {
		switch {
		case hint.Bank == "" && i > 0 && parts[i-1] == "real-pdfs":
			hint.Bank = strings.ToLower(parts[i])
		case hint.Bank != "" && hint.Layout == "" && strings.EqualFold(parts[i], hint.Bank) && i+1 < len(parts):
			hint.Layout = strings.ToLower(parts[i+1])
		}
	}

	return hint
}

func resolveOutputDirs(cfg Config, hint Hint) []string {
	root := cfg.Output
	if root == "" {
		root = "testdata"
	}

	categoryRoot := func(category string) string {
		base := strings.ToLower(filepath.Base(root))
		switch base {
		case "public-pdfs", "local-pdfs":
			if base == category {
				return root
			}
			return filepath.Join(filepath.Dir(root), category)
		default:
			return filepath.Join(root, category)
		}
	}

	basePath := func(category string) string {
		value := categoryRoot(category)
		if hint.Bank != "" {
			value = filepath.Join(value, hint.Bank)
		}
		if hint.Layout != "" {
			value = filepath.Join(value, hint.Layout)
		}
		return value
	}

	switch cfg.Mode {
	case OutputLocal:
		return []string{basePath("local-pdfs")}
	case OutputBoth:
		return []string{basePath("public-pdfs"), basePath("local-pdfs")}
	default:
		return []string{basePath("public-pdfs")}
	}
}

func validateDummy(ctx context.Context, processor *edocuenta.Processor, hint Hint, original edocuenta.ParseResult, pdfBytes []byte) (edocuenta.ParseResult, Validation) {
	result, err := processor.ParsePDFResult(ctx, pdfBytes)
	if err != nil {
		return edocuenta.ParseResult{}, Validation{
			Error: err.Error(),
		}
	}

	return result, Validation{
		ParseOK:                true,
		BankMatched:            strings.EqualFold(string(result.Statement.Bank), hint.Bank),
		LayoutMatched:          inferLayout(result) == hint.Layout || hint.Layout == "",
		TransactionsMatched:    len(result.Statement.Transactions) == len(original.Statement.Transactions),
		KindSequenceMatched:    sameKindSequence(result.Statement.Transactions, original.Statement.Transactions),
		ReferencesMatched:      sameReferencePresence(result.Statement.Transactions, original.Statement.Transactions),
		BalancePresenceMatched: sameBalancePresence(result.Statement.Transactions, original.Statement.Transactions),
	}
}

func sameKindSequence(left, right []edocuenta.Transaction) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].Direction != right[i].Direction {
			return false
		}
	}
	return true
}

func sameReferencePresence(left, right []edocuenta.Transaction) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if (left[i].Reference == "") != (right[i].Reference == "") {
			return false
		}
	}
	return true
}

func sameBalancePresence(left, right []edocuenta.Transaction) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if (left[i].BalanceCents == nil) != (right[i].BalanceCents == nil) {
			return false
		}
	}
	return true
}

func findLeaks(text string, replacements []Replacement, ctx *sanitizeContext) []string {
	leaks := make([]string, 0)
	for key, item := range ctx.replacements {
		_ = item
		parts := strings.SplitN(key, ":", 2)
		if len(parts) != 2 {
			continue
		}
		original := parts[1]
		if original != "" && strings.Contains(text, original) {
			leaks = append(leaks, hashValue(original))
		}
	}
	return leaks
}

func writeAggregateReport(path, format string, report AggregateReport) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}

	if format == "" || strings.EqualFold(format, "json") {
		return os.WriteFile(path, data, 0o600)
	}

	if strings.EqualFold(format, "markdown") || strings.EqualFold(format, "md") {
		var b strings.Builder
		b.WriteString("# fixturegen report\n\n")
		for _, file := range report.Files {
			b.WriteString(fmt.Sprintf("- `%s` -> `%s/%s` [%s]\n", file.InputPath, file.Bank, file.Layout, file.Fidelity))
		}
		return os.WriteFile(path, []byte(b.String()), 0o600)
	}

	return fmt.Errorf("unsupported report format %q", format)
}
