package pdftext

import (
	"errors"
	"fmt"
	"strings"
)

// AttemptCode classifies the outcome of an extractor attempt.
type AttemptCode string

const (
	AttemptCodeSucceeded   AttemptCode = "succeeded"
	AttemptCodeFailed      AttemptCode = "failed"
	AttemptCodeDisabled    AttemptCode = "disabled"
	AttemptCodeUnavailable AttemptCode = "unavailable"
	AttemptCodeNoText      AttemptCode = "no_text"
)

// Attempt captures one extractor result inside a chain.
type Attempt struct {
	Extractor string
	Code      AttemptCode
	Err       error
}

// ExtractorError is returned by concrete extractors so the chain can preserve
// structured failure information.
type ExtractorError struct {
	Extractor string
	Code      AttemptCode
	Err       error
}

func (e *ExtractorError) Error() string {
	if e == nil || e.Err == nil {
		return "extractor failed"
	}

	return e.Err.Error()
}

func (e *ExtractorError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Err
}

// ExtractionError aggregates all extractor attempts made by the chain.
type ExtractionError struct {
	Attempts []Attempt
}

func (e *ExtractionError) Error() string {
	if e == nil || len(e.Attempts) == 0 {
		return "extract text: no extractors configured"
	}

	parts := make([]string, 0, len(e.Attempts))
	for _, attempt := range e.Attempts {
		parts = append(parts, fmt.Sprintf("%s: %s", attempt.Extractor, describeAttempt(attempt)))
	}

	return "extract text: " + strings.Join(parts, "; ")
}

func (e *ExtractionError) Unwrap() error {
	if e == nil {
		return nil
	}

	errs := make([]error, 0, len(e.Attempts))
	for _, attempt := range e.Attempts {
		if attempt.Err != nil {
			errs = append(errs, attempt.Err)
		}
	}

	return errors.Join(errs...)
}

func newExtractorError(extractor string, code AttemptCode, err error) *ExtractorError {
	return &ExtractorError{
		Extractor: extractor,
		Code:      code,
		Err:       err,
	}
}

func noTextAttempt(extractor string) Attempt {
	return Attempt{
		Extractor: extractor,
		Code:      AttemptCodeNoText,
		Err:       errors.New("extractor returned empty text"),
	}
}

func successAttempt(extractor string) Attempt {
	return Attempt{
		Extractor: extractor,
		Code:      AttemptCodeSucceeded,
	}
}

func attemptFromError(name string, err error) Attempt {
	var extractorErr *ExtractorError
	if errors.As(err, &extractorErr) {
		return Attempt{
			Extractor: extractorErr.Extractor,
			Code:      extractorErr.Code,
			Err:       extractorErr.Err,
		}
	}

	return Attempt{
		Extractor: name,
		Code:      AttemptCodeFailed,
		Err:       err,
	}
}

func extractorName(extractor Extractor) string {
	if named, ok := extractor.(interface{ Name() string }); ok {
		return named.Name()
	}
	if extractor == nil {
		return "unknown"
	}

	return fmt.Sprintf("%T", extractor)
}

func describeAttempt(attempt Attempt) string {
	switch attempt.Code {
	case AttemptCodeDisabled:
		if attempt.Err != nil {
			return attempt.Err.Error()
		}
		return "disabled"
	case AttemptCodeSucceeded:
		return "extractor produced usable text"
	case AttemptCodeUnavailable:
		if attempt.Err != nil {
			return attempt.Err.Error()
		}
		return "required dependency not available"
	case AttemptCodeNoText:
		return "extractor returned empty text"
	default:
		if attempt.Err != nil {
			return attempt.Err.Error()
		}
		return "extractor failed"
	}
}
