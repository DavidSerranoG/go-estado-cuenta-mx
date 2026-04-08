package pdftext

import (
	"context"
	"fmt"
	"io"

	pdf "github.com/ledongthuc/pdf"
)

// Ledongthuc uses github.com/ledongthuc/pdf to extract plain text.
type Ledongthuc struct{}

// NewLedongthuc returns the default PDF extractor.
func NewLedongthuc() Ledongthuc {
	return Ledongthuc{}
}

// Name identifies the extractor in chain reports.
func (Ledongthuc) Name() string {
	return "ledongthuc"
}

// ExtractText converts a PDF into plain text.
func (Ledongthuc) ExtractText(ctx context.Context, pdfBytes []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", newExtractorError("ledongthuc", AttemptCodeFailed, err)
	}

	reader, err := pdf.NewReader(bytesReaderAt(pdfBytes), int64(len(pdfBytes)))
	if err != nil {
		return "", newExtractorError("ledongthuc", AttemptCodeFailed, fmt.Errorf("open pdf: %w", err))
	}

	plain, err := reader.GetPlainText()
	if err != nil {
		return "", newExtractorError("ledongthuc", AttemptCodeFailed, fmt.Errorf("extract plain text: %w", err))
	}

	text, err := readAllWithContext(ctx, plain)
	if err != nil {
		return "", newExtractorError("ledongthuc", AttemptCodeFailed, fmt.Errorf("read extracted text: %w", err))
	}

	return string(text), nil
}

type bytesReaderAt []byte

func (b bytesReaderAt) ReadAt(p []byte, off int64) (int, error) {
	return copyReadAt([]byte(b), p, off)
}

func readAllWithContext(ctx context.Context, reader io.Reader) ([]byte, error) {
	var data []byte
	buf := make([]byte, 32*1024)

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		n, err := reader.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err == nil {
			continue
		}
		if err == io.EOF {
			return data, nil
		}
		return nil, err
	}
}

func copyReadAt(src []byte, dst []byte, off int64) (int, error) {
	if off < 0 {
		return 0, fmt.Errorf("negative offset")
	}
	if off >= int64(len(src)) {
		return 0, io.EOF
	}

	n := copy(dst, src[off:])
	if n < len(dst) {
		return n, io.EOF
	}
	return n, nil
}
