package pdftext

import (
	"context"
	"errors"
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
		NewPdftotext(),
	)
}

// ExtractCandidates evaluates every extractor in order and returns each usable
// text candidate together with all typed attempts.
func (c Chain) ExtractCandidates(ctx context.Context, pdfBytes []byte) (CandidateRun, error) {
	if err := ctx.Err(); err != nil {
		return CandidateRun{}, err
	}

	run := CandidateRun{
		Candidates: make([]Candidate, 0, len(c.extractors)),
		Attempts:   make([]Attempt, 0, len(c.extractors)),
	}

	for _, extractor := range c.extractors {
		if extractor == nil {
			continue
		}
		if err := ctx.Err(); err != nil {
			return run, err
		}

		name := extractorName(extractor)
		text, err := extractor.ExtractText(ctx, pdfBytes)
		switch {
		case err == nil && strings.TrimSpace(text) != "":
			run.Candidates = append(run.Candidates, NewCandidate(name, text))
			run.Attempts = append(run.Attempts, successAttempt(name))
		case err == nil:
			run.Attempts = append(run.Attempts, noTextAttempt(name))
		default:
			run.Attempts = append(run.Attempts, attemptFromError(name, err))
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return run, nil
			}
		}
	}

	return run, nil
}

// ExtractText returns the first successful extraction result.
func (c Chain) ExtractText(ctx context.Context, pdfBytes []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	attempts := make([]Attempt, 0, len(c.extractors))

	for _, extractor := range c.extractors {
		if extractor == nil {
			continue
		}
		if err := ctx.Err(); err != nil {
			return "", err
		}

		name := extractorName(extractor)
		text, err := extractor.ExtractText(ctx, pdfBytes)
		if err == nil && strings.TrimSpace(text) != "" {
			return text, nil
		}
		if err == nil {
			attempts = append(attempts, noTextAttempt(name))
			continue
		}

		attempts = append(attempts, attemptFromError(name, err))
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			break
		}
	}

	if len(attempts) == 0 {
		return "", &ExtractionError{}
	}

	return "", &ExtractionError{Attempts: attempts}
}
