package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	edocuenta "github.com/DavidSerranoG/go-estado-cuenta-mx"
	"github.com/DavidSerranoG/go-estado-cuenta-mx/supported"
)

type fileReport struct {
	Path               string `json:"path"`
	ExpectedBank       string `json:"expected_bank"`
	ExpectedLayout     string `json:"expected_layout"`
	DetectedBank       string `json:"detected_bank,omitempty"`
	ExtractionNonEmpty bool   `json:"extraction_non_empty"`
	ParseOK            bool   `json:"parse_ok"`
	BankMatched        bool   `json:"bank_matched"`
	SelectedExtractor  string `json:"selected_extractor,omitempty"`
	UsedRescue         bool   `json:"used_rescue"`
	Transactions       int    `json:"transactions"`
	Warnings           int    `json:"warnings"`
	Error              string `json:"error,omitempty"`
}

type bucketSummary struct {
	Bank         string `json:"bank"`
	Layout       string `json:"layout"`
	Files        int    `json:"files"`
	Extracted    int    `json:"extracted"`
	Parsed       int    `json:"parsed"`
	BankMatched  int    `json:"bank_matched"`
	Transactions int    `json:"transactions"`
	Warnings     int    `json:"warnings"`
}

type summary struct {
	Files        int             `json:"files"`
	Extracted    int             `json:"extracted"`
	Parsed       int             `json:"parsed"`
	BankMatched  int             `json:"bank_matched"`
	Transactions int             `json:"transactions"`
	Warnings     int             `json:"warnings"`
	ByBucket     []bucketSummary `json:"by_bucket"`
}

type report struct {
	Root    string       `json:"root"`
	Rescue  string       `json:"rescue"`
	Files   []fileReport `json:"files"`
	Summary summary      `json:"summary"`
}

func main() {
	root := flag.String("root", ".tmp/real-pdfs", "directory containing private real-pdf corpora")
	rescue := flag.String("rescue", "none", "optional rescue extractor: none, tesseract, or vision")
	format := flag.String("format", "json", "output format: json or markdown")
	flag.Parse()

	processor, rescueName, err := newProcessor(strings.ToLower(strings.TrimSpace(*rescue)))
	if err != nil {
		fail(err)
	}

	reports, err := evaluateCorpus(*root, processor)
	if err != nil {
		fail(err)
	}

	output := report{
		Root:    *root,
		Rescue:  rescueName,
		Files:   reports,
		Summary: buildSummary(reports),
	}

	switch strings.ToLower(strings.TrimSpace(*format)) {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(output); err != nil {
			fail(err)
		}
	case "markdown", "md":
		if err := renderMarkdown(os.Stdout, output); err != nil {
			fail(err)
		}
	default:
		fail(fmt.Errorf("unsupported format %q", *format))
	}
}

func newProcessor(rescue string) (*edocuenta.Processor, string, error) {
	opts := []edocuenta.Option{}

	switch rescue {
	case "", "none":
		return supported.New(opts...), "none", nil
	case "tesseract":
		opts = append(opts, edocuenta.WithRescueExtractor(edocuenta.NewTesseractExtractor()))
		return supported.New(opts...), "tesseract", nil
	case "vision":
		opts = append(opts, edocuenta.WithRescueExtractor(edocuenta.NewVisionExtractor()))
		return supported.New(opts...), "vision", nil
	default:
		return nil, "", fmt.Errorf("unsupported rescue extractor %q", rescue)
	}
}

func evaluateCorpus(root string, processor *edocuenta.Processor) ([]fileReport, error) {
	files := make([]string, 0)
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".pdf") {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	sort.Strings(files)

	reports := make([]fileReport, 0, len(files))
	for _, path := range files {
		report, err := evaluateFile(root, path, processor)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}

	return reports, nil
}

func evaluateFile(root, path string, processor *edocuenta.Processor) (fileReport, error) {
	expectedBank, expectedLayout := bucketFromPath(root, path)
	bytes, err := os.ReadFile(path)
	if err != nil {
		return fileReport{}, err
	}

	result, parseErr := processor.ParsePDFResult(context.Background(), bytes)
	relativePath, err := filepath.Rel(root, path)
	if err != nil {
		relativePath = path
	}

	report := fileReport{
		Path:               filepath.ToSlash(relativePath),
		ExpectedBank:       expectedBank,
		ExpectedLayout:     expectedLayout,
		ExtractionNonEmpty: strings.TrimSpace(result.ExtractedText) != "",
		SelectedExtractor:  result.Extraction.SelectedExtractor,
		UsedRescue:         result.Extraction.UsedRescue,
		Transactions:       len(result.Statement.Transactions),
		Warnings:           len(result.Warnings),
	}

	if parseErr != nil {
		report.Error = parseErr.Error()
		return report, nil
	}

	report.ParseOK = true
	report.DetectedBank = string(result.Statement.Bank)
	report.BankMatched = expectedBank == "" || strings.EqualFold(expectedBank, string(result.Statement.Bank))
	return report, nil
}

func bucketFromPath(root, path string) (string, string) {
	relativePath, err := filepath.Rel(root, path)
	if err != nil {
		return "", "unknown"
	}

	parts := strings.Split(filepath.ToSlash(relativePath), "/")
	if len(parts) >= 3 {
		return strings.ToLower(parts[0]), strings.ToLower(parts[1])
	}
	if len(parts) >= 2 {
		return strings.ToLower(parts[0]), "unknown"
	}
	return "", "unknown"
}

func buildSummary(files []fileReport) summary {
	buckets := map[string]*bucketSummary{}
	result := summary{Files: len(files)}

	for _, file := range files {
		if file.ExtractionNonEmpty {
			result.Extracted++
		}
		if file.ParseOK {
			result.Parsed++
			result.Transactions += file.Transactions
			result.Warnings += file.Warnings
		}
		if file.BankMatched {
			result.BankMatched++
		}

		key := file.ExpectedBank + "::" + file.ExpectedLayout
		bucket := buckets[key]
		if bucket == nil {
			bucket = &bucketSummary{
				Bank:   file.ExpectedBank,
				Layout: file.ExpectedLayout,
			}
			buckets[key] = bucket
		}

		bucket.Files++
		if file.ExtractionNonEmpty {
			bucket.Extracted++
		}
		if file.ParseOK {
			bucket.Parsed++
			bucket.Transactions += file.Transactions
			bucket.Warnings += file.Warnings
		}
		if file.BankMatched {
			bucket.BankMatched++
		}
	}

	keys := make([]string, 0, len(buckets))
	for key := range buckets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		result.ByBucket = append(result.ByBucket, *buckets[key])
	}

	return result
}

func renderMarkdown(output *os.File, report report) error {
	if _, err := fmt.Fprintf(output, "# edocuenta eval\n\n- Root: `%s`\n- Rescue: `%s`\n- Files: %d\n- Extracted: %d\n- Parsed: %d\n- Bank matched: %d\n\n", report.Root, report.Rescue, report.Summary.Files, report.Summary.Extracted, report.Summary.Parsed, report.Summary.BankMatched); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(output, "## By bucket"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(output, "\n| Bank | Layout | Files | Extracted | Parsed | Bank matched | Tx | Warnings |"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(output, "| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |"); err != nil {
		return err
	}
	for _, bucket := range report.Summary.ByBucket {
		if _, err := fmt.Fprintf(output, "| %s | %s | %d | %d | %d | %d | %d | %d |\n", bucket.Bank, bucket.Layout, bucket.Files, bucket.Extracted, bucket.Parsed, bucket.BankMatched, bucket.Transactions, bucket.Warnings); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(output, "\n## Files"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(output, "\n| Path | Bank | Layout | Parse | Extractor | Rescue | Tx | Warnings |"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(output, "| --- | --- | --- | --- | --- | --- | ---: | ---: |"); err != nil {
		return err
	}
	for _, file := range report.Files {
		status := "no"
		if file.ParseOK {
			status = "yes"
		}
		if _, err := fmt.Fprintf(output, "| %s | %s | %s | %s | %s | %t | %d | %d |\n", file.Path, file.ExpectedBank, file.ExpectedLayout, status, file.SelectedExtractor, file.UsedRescue, file.Transactions, file.Warnings); err != nil {
			return err
		}
	}

	return nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
