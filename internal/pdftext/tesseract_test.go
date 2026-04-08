package pdftext

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

func TestTesseractReportsUnavailableBinary(t *testing.T) {
	t.Parallel()

	extractor := Tesseract{
		goos: "linux",
		lookPath: func(name string) (string, error) {
			if name == "gs" {
				return "", exec.ErrNotFound
			}
			return "/usr/bin/" + name, nil
		},
	}

	_, err := extractor.ExtractText(context.Background(), []byte("pdf"))
	if err == nil {
		t.Fatal("expected extractor error")
	}

	var extractorErr *ExtractorError
	if !errors.As(err, &extractorErr) {
		t.Fatalf("expected ExtractorError, got %T", err)
	}
	if extractorErr.Code != AttemptCodeUnavailable {
		t.Fatalf("unexpected attempt code %q", extractorErr.Code)
	}
}

func TestTesseractExtractsStubbedText(t *testing.T) {
	t.Parallel()

	extractor := Tesseract{
		lookPath: func(name string) (string, error) {
			return "/usr/bin/" + name, nil
		},
		rasterizeGhostscript: func(context.Context, string, []byte) (string, []string, error) {
			return "", []string{"page-1.png", "page-2.png"}, nil
		},
		ocrPages: func(context.Context, string, []string) (string, error) {
			return "BBVA\nTARJETA", nil
		},
	}

	text, err := extractor.ExtractText(context.Background(), []byte("pdf"))
	if err != nil {
		t.Fatalf("extract text: %v", err)
	}
	if text != "BBVA\nTARJETA" {
		t.Fatalf("unexpected text %q", text)
	}
}

func TestTesseractFallsBackToDarwinPDFKitWhenGhostscriptMissing(t *testing.T) {
	t.Parallel()

	var usedPDFKit bool
	extractor := Tesseract{
		goos: "darwin",
		lookPath: func(name string) (string, error) {
			if name == "gs" {
				return "", exec.ErrNotFound
			}
			return "/usr/bin/" + name, nil
		},
		rasterizeDarwinPDFKit: func(context.Context, string, []byte) (string, []string, error) {
			usedPDFKit = true
			return "", []string{"page-1.png"}, nil
		},
		ocrPages: func(context.Context, string, []string) (string, error) {
			return "HSBC", nil
		},
	}

	text, err := extractor.ExtractText(context.Background(), []byte("pdf"))
	if err != nil {
		t.Fatalf("extract text: %v", err)
	}
	if !usedPDFKit {
		t.Fatal("expected darwin PDFKit rasterizer to be used")
	}
	if text != "HSBC" {
		t.Fatalf("unexpected text %q", text)
	}
}

func TestTesseractHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewTesseract().ExtractText(ctx, []byte("pdf"))
	if err == nil {
		t.Fatal("expected extractor error")
	}

	var extractorErr *ExtractorError
	if !errors.As(err, &extractorErr) {
		t.Fatalf("expected ExtractorError, got %T", err)
	}
	if !errors.Is(extractorErr.Err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", extractorErr.Err)
	}
}
