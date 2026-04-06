package pdftext

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	pdf "github.com/ledongthuc/pdf"
)

// Ledongthuc uses github.com/ledongthuc/pdf to extract plain text.
type Ledongthuc struct{}

// NewLedongthuc returns the default PDF extractor.
func NewLedongthuc() Ledongthuc {
	return Ledongthuc{}
}

// ExtractText converts a PDF into plain text.
func (Ledongthuc) ExtractText(_ context.Context, pdfBytes []byte) (string, error) {
	tmpFile, err := os.CreateTemp("", "mxstatementpdf-*.pdf")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := io.Copy(tmpFile, bytes.NewReader(pdfBytes)); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("write temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("close temp file: %w", err)
	}

	file, reader, err := pdf.Open(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("open pdf: %w", err)
	}
	defer file.Close()

	plain, err := reader.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("extract plain text: %w", err)
	}

	text, err := io.ReadAll(plain)
	if err != nil {
		return "", fmt.Errorf("read extracted text: %w", err)
	}

	return string(text), nil
}
