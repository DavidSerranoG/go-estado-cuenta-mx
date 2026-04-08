package pdftext

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

func TestPdftotextReportsUnavailableBinary(t *testing.T) {
	t.Parallel()

	extractor := Pdftotext{
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

func TestPdftotextHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewPdftotext().ExtractText(ctx, []byte("pdf"))
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
