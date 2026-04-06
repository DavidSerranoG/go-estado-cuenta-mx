package pdftext

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Chain tries multiple extractors in order until one succeeds with non-empty text.
type Chain struct {
	extractors []Extractor
}

// NewChain creates an extractor chain.
func NewChain(extractors ...Extractor) Chain {
	return Chain{extractors: extractors}
}

// NewDefault returns the default extractor chain for the package.
func NewDefault() Chain {
	return NewChain(
		NewLedongthuc(),
		NewPyPDF(""),
	)
}

// ExtractText returns the first successful extraction result.
func (c Chain) ExtractText(ctx context.Context, pdfBytes []byte) (string, error) {
	var errs []error

	for _, extractor := range c.extractors {
		if extractor == nil {
			continue
		}

		text, err := extractor.ExtractText(ctx, pdfBytes)
		if err == nil && strings.TrimSpace(text) != "" {
			return text, nil
		}
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) == 0 {
		return "", fmt.Errorf("extract text: no extractor produced output")
	}

	return "", errors.Join(errs...)
}
