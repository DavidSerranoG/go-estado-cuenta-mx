package pdftext

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Pdftotext uses the poppler pdftotext binary as an optional fallback extractor.
type Pdftotext struct {
	lookPath func(string) (string, error)
	run      func(context.Context, string, []byte) (string, error)
}

// NewPdftotext returns the optional pdftotext-based extractor.
func NewPdftotext() Pdftotext {
	return Pdftotext{}
}

// Name identifies the extractor in chain reports.
func (Pdftotext) Name() string {
	return "pdftotext"
}

// ExtractText converts a PDF into plain text using the pdftotext binary.
func (p Pdftotext) ExtractText(ctx context.Context, pdfBytes []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", newExtractorError("pdftotext", AttemptCodeFailed, err)
	}

	lookPath := p.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	binary, err := lookPath("pdftotext")
	if err != nil {
		return "", newExtractorError("pdftotext", AttemptCodeUnavailable, fmt.Errorf("find pdftotext binary: %w", err))
	}

	run := p.run
	if run == nil {
		run = runPdftotext
	}

	text, err := run(ctx, binary, pdfBytes)
	if err != nil {
		return "", newExtractorError("pdftotext", AttemptCodeFailed, err)
	}

	return text, nil
}

func runPdftotext(ctx context.Context, binary string, pdfBytes []byte) (string, error) {
	input, err := os.CreateTemp("", "edocuenta-*.pdf")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(input.Name())

	if _, err := input.Write(pdfBytes); err != nil {
		input.Close()
		return "", fmt.Errorf("write temp file: %w", err)
	}
	if err := input.Close(); err != nil {
		return "", fmt.Errorf("close temp file: %w", err)
	}

	output, err := os.CreateTemp("", "edocuenta-*.txt")
	if err != nil {
		return "", fmt.Errorf("create output file: %w", err)
	}
	outputPath := output.Name()
	if err := output.Close(); err != nil {
		return "", fmt.Errorf("close output file: %w", err)
	}
	defer os.Remove(outputPath)

	cmd := exec.CommandContext(ctx, binary, "-layout", "-nopgbrk", "-q", input.Name(), outputPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		if stderr.Len() > 0 {
			return "", fmt.Errorf("run pdftotext: %s", strings.TrimSpace(stderr.String()))
		}
		return "", fmt.Errorf("run pdftotext: %w", err)
	}

	text, err := os.ReadFile(outputPath)
	if err != nil {
		return "", fmt.Errorf("read pdftotext output: %w", err)
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return "", ctxErr
	}

	trimmed := strings.ReplaceAll(string(text), "\x00", "")
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return "", nil
	}

	return trimmed, nil
}
