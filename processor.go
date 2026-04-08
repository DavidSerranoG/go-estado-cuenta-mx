package edocuenta

import (
	"context"

	"github.com/DavidSerranoG/go-estado-cuenta-mx/internal/registry"
)

// TextExtractor extracts plain text from a PDF payload.
type TextExtractor interface {
	ExtractText(ctx context.Context, pdfBytes []byte) (string, error)
}

// Parser knows how to detect and parse a specific bank statement layout.
type Parser interface {
	Bank() string
	CanParse(text string) bool
	Parse(text string) (Statement, error)
}

// ResultParser optionally exposes parser warnings and other parsing diagnostics.
type ResultParser interface {
	ParseResult(text string) (ParseResult, error)
}

// ScoredParser optionally exposes stronger bank detection.
//
// When multiple parsers are registered, the processor prefers the parser with
// the highest positive score. Parsers that do not implement this interface
// still participate using their boolean CanParse result.
type ScoredParser interface {
	DetectionScore(text string) int
}

// Processor orchestrates PDF text extraction and bank-specific parsing.
type Processor struct {
	extractor       TextExtractor
	rescueExtractor TextExtractor
	parsers         *registry.Store[Parser]
}

// New builds a Processor with the provided options.
func New(opts ...Option) *Processor {
	processor := &Processor{
		parsers: registry.New[Parser](),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(processor)
		}
	}

	if processor.extractor == nil {
		processor.extractor = NewDefaultTextExtractor()
	}

	return processor
}

// ParsePDF extracts text from the PDF, auto-detects the parser, and returns the normalized statement.
func (p *Processor) ParsePDF(ctx context.Context, pdfBytes []byte) (Statement, error) {
	result, err := p.ParsePDFResult(ctx, pdfBytes)
	if err != nil {
		return Statement{}, err
	}

	return result.Statement, nil
}

// ParsePDFResult extracts text from the PDF, auto-detects the parser, and
// returns the normalized statement plus parse diagnostics.
func (p *Processor) ParsePDFResult(ctx context.Context, pdfBytes []byte) (ParseResult, error) {
	return p.parsePDFResult(ctx, pdfBytes, "")
}

// ParsePDFWithBank extracts text and uses an explicit parser by bank identifier.
func (p *Processor) ParsePDFWithBank(ctx context.Context, pdfBytes []byte, bank string) (Statement, error) {
	result, err := p.ParsePDFWithBankResult(ctx, pdfBytes, bank)
	if err != nil {
		return Statement{}, err
	}

	return result.Statement, nil
}

// ParsePDFWithBankResult extracts text and uses an explicit parser by bank
// identifier, returning diagnostics as well.
func (p *Processor) ParsePDFWithBankResult(ctx context.Context, pdfBytes []byte, bank string) (ParseResult, error) {
	return p.parsePDFResult(ctx, pdfBytes, bank)
}

// ParseText auto-detects the parser from extracted plain text.
func (p *Processor) ParseText(text string) (Statement, error) {
	result, err := p.ParseTextResult(text)
	if err != nil {
		return Statement{}, err
	}

	return result.Statement, nil
}

// ParseTextResult auto-detects the parser from extracted plain text and
// returns the normalized statement plus parser diagnostics.
func (p *Processor) ParseTextResult(text string) (ParseResult, error) {
	return p.parseTextCandidate(text, "")
}

// ParseTextWithBank uses an explicit parser by bank identifier.
func (p *Processor) ParseTextWithBank(text string, bank string) (Statement, error) {
	result, err := p.ParseTextWithBankResult(text, bank)
	if err != nil {
		return Statement{}, err
	}

	return result.Statement, nil
}

// ParseTextWithBankResult uses an explicit parser by bank identifier and
// returns the normalized statement plus parser diagnostics.
func (p *Processor) ParseTextWithBankResult(text string, bank string) (ParseResult, error) {
	return p.parseTextCandidate(text, bank)
}

func (p *Processor) parseWithParser(parser Parser, text string) (ParseResult, error) {
	if detailed, ok := parser.(ResultParser); ok {
		return detailed.ParseResult(text)
	}

	statement, err := parser.Parse(text)
	if err != nil {
		return ParseResult{}, err
	}

	return ParseResult{Statement: statement}, nil
}

func (p *Processor) byBank(bank string) (Parser, error) {
	if p.parsers.Len() == 0 {
		return nil, ErrNoParsersConfigured
	}

	parser, ok := p.parsers.FindByBank(bank)
	if !ok {
		return nil, ErrUnsupportedBank
	}

	return parser, nil
}
