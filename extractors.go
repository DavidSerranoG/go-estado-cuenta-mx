package edocuenta

import "github.com/DavidSerranoG/go-estado-cuenta-mx/internal/pdftext"

// NewDefaultTextExtractor returns the lightweight extractor chain used by New.
//
// The default path favors predictable dependencies for library consumers and
// does not enable OCR automatically.
func NewDefaultTextExtractor() TextExtractor {
	return NewTextExtractorChain(
		NewLedongthucExtractor(),
		NewPdftotextExtractor(),
	)
}

// NewTextExtractorChain builds a TextExtractor that tries each extractor in
// order until one returns usable text.
func NewTextExtractorChain(extractors ...TextExtractor) TextExtractor {
	internalExtractors := make([]pdftext.Extractor, 0, len(extractors))
	for _, extractor := range extractors {
		if extractor != nil {
			internalExtractors = append(internalExtractors, extractor)
		}
	}

	return pdftext.NewChain(internalExtractors...)
}

// NewLedongthucExtractor returns the built-in `ledongthuc`-based text extractor.
func NewLedongthucExtractor() TextExtractor {
	return pdftext.NewLedongthuc()
}

// NewPdftotextExtractor returns the built-in pdftotext-based fallback extractor.
//
// This extractor is only usable when the `pdftotext` binary is available on the
// host machine.
func NewPdftotextExtractor() TextExtractor {
	return pdftext.NewPdftotext()
}

// NewVisionExtractor returns the built-in macOS PDFKit + Vision OCR extractor.
//
// This extractor is only usable on macOS when the `swift` runtime is available.
func NewVisionExtractor() TextExtractor {
	return pdftext.NewVision()
}

// NewTesseractExtractor returns the built-in tesseract-based OCR extractor.
//
// This extractor rasterizes PDF pages with `gs` and runs `tesseract` on each
// page. It is intended primarily as an optional rescue path for image-heavy or
// degraded statements.
func NewTesseractExtractor() TextExtractor {
	return pdftext.NewTesseract()
}

// Deprecated: use NewLedongthucExtractor.
func NewGoTextExtractor() TextExtractor {
	return NewLedongthucExtractor()
}

// Deprecated: use NewVisionExtractor.
func NewMacOSVisionExtractor() TextExtractor {
	return NewVisionExtractor()
}

// Deprecated: use NewTesseractExtractor.
func NewTesseractOCRExtractor() TextExtractor {
	return NewTesseractExtractor()
}
