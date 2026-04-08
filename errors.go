package edocuenta

import (
	"errors"
	"strings"

	"github.com/DavidSerranoG/go-estado-cuenta-mx/internal/pdftext"
)

var (
	ErrEmptyPDF            = errors.New("edocuenta: empty pdf payload")
	ErrNoParsersConfigured = errors.New("edocuenta: no parsers configured")
	ErrUnsupportedFormat   = errors.New("edocuenta: unsupported statement format")
	ErrUnsupportedBank     = errors.New("edocuenta: unsupported bank")
	ErrTextExtraction      = errors.New("edocuenta: pdf text extraction failed")
	ErrLedongthucExtractor = errors.New("edocuenta: ledongthuc extractor failed")
	ErrPdftotextExtractor  = errors.New("edocuenta: pdftotext extractor failed")
	ErrVisionExtractor     = errors.New("edocuenta: vision extractor failed")
	ErrTesseractExtractor  = errors.New("edocuenta: tesseract extractor failed")
	ErrNoUsableText        = errors.New("edocuenta: no extractor produced usable text")
)

// Deprecated: use ErrLedongthucExtractor.
var ErrGoTextExtractor = ErrLedongthucExtractor

// Deprecated: use ErrVisionExtractor.
var ErrMacOSVisionExtractor = ErrVisionExtractor

// TextExtractionAttemptStatus describes the outcome of a single extractor attempt.
type TextExtractionAttemptStatus string

const (
	TextExtractionAttemptSucceeded   TextExtractionAttemptStatus = "succeeded"
	TextExtractionAttemptFailed      TextExtractionAttemptStatus = "failed"
	TextExtractionAttemptDisabled    TextExtractionAttemptStatus = "disabled"
	TextExtractionAttemptUnavailable TextExtractionAttemptStatus = "unavailable"
	TextExtractionAttemptNoText      TextExtractionAttemptStatus = "no_text"
)

// TextExtractionAttempt is a normalized, user-facing view of one extractor attempt.
type TextExtractionAttempt struct {
	Extractor string
	Status    TextExtractionAttemptStatus
	Message   string
}

// TextExtractionError exposes structured information about all extractor attempts.
type TextExtractionError struct {
	Attempts []TextExtractionAttempt
	cause    error
}

// Error returns a readable summary of the extraction failure.
func (e *TextExtractionError) Error() string {
	if e == nil {
		return ErrTextExtraction.Error()
	}
	if len(e.Attempts) == 0 {
		return ErrTextExtraction.Error()
	}

	var b strings.Builder
	b.WriteString(ErrTextExtraction.Error())
	for _, attempt := range e.Attempts {
		b.WriteString("\n- ")
		b.WriteString(attempt.Extractor)
		b.WriteString(": ")
		b.WriteString(attempt.Message)
	}

	return b.String()
}

// Unwrap exposes sentinel and underlying extractor errors via errors.Is / errors.As.
func (e *TextExtractionError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.cause
}

func wrapTextExtractionError(err error) error {
	var extractionErr *pdftext.ExtractionError
	if !errors.As(err, &extractionErr) {
		return err
	}

	return newTextExtractionError(extractionErr.Attempts)
}

func newTextExtractionError(attemptList []pdftext.Attempt) *TextExtractionError {
	attempts := make([]TextExtractionAttempt, 0, len(attemptList))
	causes := []error{ErrTextExtraction}
	noTextOnly := len(attemptList) > 0

	for _, attempt := range attemptList {
		attempts = append(attempts, publicAttempt(attempt))

		if attempt.Err != nil {
			causes = append(causes, attempt.Err)
		}

		switch attempt.Code {
		case pdftext.AttemptCodeSucceeded:
			noTextOnly = false
		case pdftext.AttemptCodeFailed:
			noTextOnly = false
			if sentinel := extractorSentinel(attempt.Extractor); sentinel != nil {
				causes = append(causes, sentinel)
			}
		case pdftext.AttemptCodeDisabled:
			noTextOnly = false
		case pdftext.AttemptCodeUnavailable:
			noTextOnly = false
		case pdftext.AttemptCodeNoText:
			causes = append(causes, ErrNoUsableText)
		}
	}

	if noTextOnly {
		causes = append(causes, ErrNoUsableText)
	}

	return &TextExtractionError{
		Attempts: attempts,
		cause:    errors.Join(causes...),
	}
}

func publicAttempts(attemptList []pdftext.Attempt) []TextExtractionAttempt {
	attempts := make([]TextExtractionAttempt, 0, len(attemptList))
	for _, attempt := range attemptList {
		attempts = append(attempts, publicAttempt(attempt))
	}
	return attempts
}

func publicAttempt(attempt pdftext.Attempt) TextExtractionAttempt {
	return TextExtractionAttempt{
		Extractor: attempt.Extractor,
		Status:    mapAttemptStatus(attempt.Code),
		Message:   describeAttempt(attempt),
	}
}

func extractorSentinel(name string) error {
	switch name {
	case "ledongthuc":
		return ErrLedongthucExtractor
	case "pdftotext":
		return ErrPdftotextExtractor
	case "vision":
		return ErrVisionExtractor
	case "tesseract":
		return ErrTesseractExtractor
	default:
		return nil
	}
}

func mapAttemptStatus(code pdftext.AttemptCode) TextExtractionAttemptStatus {
	switch code {
	case pdftext.AttemptCodeSucceeded:
		return TextExtractionAttemptSucceeded
	case pdftext.AttemptCodeDisabled:
		return TextExtractionAttemptDisabled
	case pdftext.AttemptCodeUnavailable:
		return TextExtractionAttemptUnavailable
	case pdftext.AttemptCodeNoText:
		return TextExtractionAttemptNoText
	default:
		return TextExtractionAttemptFailed
	}
}

func describeAttempt(attempt pdftext.Attempt) string {
	switch attempt.Code {
	case pdftext.AttemptCodeDisabled:
		if attempt.Err != nil {
			return attempt.Err.Error()
		}
		return "extractor is disabled"
	case pdftext.AttemptCodeUnavailable:
		if attempt.Err != nil {
			return attempt.Err.Error()
		}
		return "required dependency is not available"
	case pdftext.AttemptCodeNoText:
		return "extractor completed but returned no usable text"
	default:
		if attempt.Err != nil {
			return attempt.Err.Error()
		}
		return "extractor failed"
	}
}
