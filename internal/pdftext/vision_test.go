package pdftext

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

func TestVisionReportsUnavailableOffDarwin(t *testing.T) {
	t.Parallel()

	extractor := Vision{
		goos: "linux",
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

func TestVisionReportsUnavailableSwiftBinary(t *testing.T) {
	t.Parallel()

	extractor := Vision{
		goos: "darwin",
		lookPath: func(string) (string, error) {
			return "", exec.ErrNotFound
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

func TestVisionExtractsStubbedText(t *testing.T) {
	t.Parallel()

	extractor := Vision{
		goos: "darwin",
		lookPath: func(string) (string, error) {
			return "/usr/bin/swift", nil
		},
		run: func(context.Context, string, []byte) (string, error) {
			return "HSBC\nNUMERO DE CUENTA", nil
		},
	}

	text, err := extractor.ExtractText(context.Background(), []byte("pdf"))
	if err != nil {
		t.Fatalf("extract text: %v", err)
	}
	if text != "HSBC\nNUMERO DE CUENTA" {
		t.Fatalf("unexpected text %q", text)
	}
}
