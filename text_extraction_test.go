package edocuenta_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	edocuenta "github.com/DavidSerranoG/go-estado-cuenta-mx"
)

func TestProcessorDefaultExtractionReportsLedongthucAttempt(t *testing.T) {
	t.Parallel()

	processor := edocuenta.New()
	_, err := processor.ParsePDF(context.Background(), []byte("not a pdf"))
	if err == nil {
		t.Fatal("expected extraction error")
	}
	if !errors.Is(err, edocuenta.ErrLedongthucExtractor) {
		t.Fatalf("expected ledongthuc extractor failure, got %v", err)
	}

	var extractionErr *edocuenta.TextExtractionError
	if !errors.As(err, &extractionErr) {
		t.Fatalf("expected TextExtractionError, got %T", err)
	}
	if len(extractionErr.Attempts) == 0 {
		t.Fatal("expected at least one extraction attempt")
	}
	if extractionErr.Attempts[0].Extractor != "ledongthuc" || extractionErr.Attempts[0].Status != edocuenta.TextExtractionAttemptFailed {
		t.Fatalf("unexpected first attempt %+v", extractionErr.Attempts[0])
	}
	for _, attempt := range extractionErr.Attempts {
		if attempt.Extractor == "tesseract" || attempt.Extractor == "vision" {
			t.Fatalf("default extraction path should not use OCR, got %+v", attempt)
		}
	}
}

func TestProcessorUsesCustomExtractorOverride(t *testing.T) {
	t.Parallel()

	processor := edocuenta.New(
		edocuenta.WithExtractor(staticExtractor{text: "FAKE BANK\n"}),
		edocuenta.WithParser(matchingParser{
			bank:  "fake",
			match: "FAKE BANK",
			statement: edocuenta.Statement{
				Bank:         "fake",
				Transactions: []edocuenta.Transaction{{Description: "fake"}},
			},
		}),
	)

	result, err := processor.ParsePDFResult(context.Background(), []byte("not a pdf"))
	if err != nil {
		t.Fatalf("parse pdf with explicit extractor: %v", err)
	}
	statement := result.Statement
	if statement.Bank != "fake" {
		t.Fatalf("expected fake bank, got %q", statement.Bank)
	}
	if result.ExtractedText != "FAKE BANK\n" {
		t.Fatalf("unexpected extracted text %q", result.ExtractedText)
	}
	if result.Extraction.SelectedExtractor != "static" {
		t.Fatalf("unexpected selected extractor %q", result.Extraction.SelectedExtractor)
	}
}

func TestPublicExtractorChainUsesFirstUsableText(t *testing.T) {
	t.Parallel()

	extractor := edocuenta.NewTextExtractorChain(
		staticExtractor{text: "   "},
		staticExtractor{text: "BBVA\n"},
	)

	text, err := extractor.ExtractText(context.Background(), []byte("pdf"))
	if err != nil {
		t.Fatalf("extract text: %v", err)
	}
	if text != "BBVA\n" {
		t.Fatalf("unexpected text %q", text)
	}
}

type staticExtractor struct {
	text string
}

func (e staticExtractor) Name() string {
	return "static"
}

func (e staticExtractor) ExtractText(context.Context, []byte) (string, error) {
	return e.text, nil
}

type matchingParser struct {
	bank      string
	match     string
	statement edocuenta.Statement
}

func (p matchingParser) Bank() string {
	return p.bank
}

func (p matchingParser) CanParse(text string) bool {
	return strings.Contains(text, p.match)
}

func (p matchingParser) Parse(string) (edocuenta.Statement, error) {
	return p.statement, nil
}
