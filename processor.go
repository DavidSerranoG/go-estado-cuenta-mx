package statementpdf

import (
	"context"

	"github.com/ledgermx/mxstatementpdf/internal/pdftext"
	"github.com/ledgermx/mxstatementpdf/internal/registry"
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

// Processor orchestrates PDF text extraction and bank-specific parsing.
type Processor struct {
	extractor TextExtractor
	parsers   *registry.Store[Parser]
}

// New builds a Processor with the provided options.
func New(opts ...Option) *Processor {
	processor := &Processor{
		extractor: pdftext.NewDefault(),
		parsers:   registry.New[Parser](),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(processor)
		}
	}

	return processor
}

// ParsePDF extracts text from the PDF, auto-detects the parser, and returns the normalized statement.
func (p *Processor) ParsePDF(ctx context.Context, pdfBytes []byte) (Statement, error) {
	if len(pdfBytes) == 0 {
		return Statement{}, ErrEmptyPDF
	}

	text, err := p.extractor.ExtractText(ctx, pdfBytes)
	if err != nil {
		return Statement{}, err
	}

	return p.ParseText(text)
}

// ParsePDFWithBank extracts text and uses an explicit parser by bank identifier.
func (p *Processor) ParsePDFWithBank(ctx context.Context, pdfBytes []byte, bank string) (Statement, error) {
	if len(pdfBytes) == 0 {
		return Statement{}, ErrEmptyPDF
	}

	text, err := p.extractor.ExtractText(ctx, pdfBytes)
	if err != nil {
		return Statement{}, err
	}

	return p.ParseTextWithBank(text, bank)
}

// ParseText auto-detects the parser from extracted plain text.
func (p *Processor) ParseText(text string) (Statement, error) {
	parser, err := p.detect(text)
	if err != nil {
		return Statement{}, err
	}

	statement, err := parser.Parse(text)
	if err != nil {
		return Statement{}, err
	}

	statement.ExtractedText = text
	return statement, nil
}

// ParseTextWithBank uses an explicit parser by bank identifier.
func (p *Processor) ParseTextWithBank(text string, bank string) (Statement, error) {
	parser, err := p.byBank(bank)
	if err != nil {
		return Statement{}, err
	}

	statement, err := parser.Parse(text)
	if err != nil {
		return Statement{}, err
	}

	statement.ExtractedText = text
	return statement, nil
}

func (p *Processor) detect(text string) (Parser, error) {
	if p.parsers.Len() == 0 {
		return nil, ErrNoParsersConfigured
	}

	parser, ok := p.parsers.FindByText(text)
	if !ok {
		return nil, ErrUnsupportedFormat
	}

	return parser, nil
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
