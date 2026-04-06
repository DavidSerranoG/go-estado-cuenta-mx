package pdftext

import "context"

// Extractor extracts plain text from raw PDF bytes.
type Extractor interface {
	ExtractText(ctx context.Context, pdfBytes []byte) (string, error)
}
