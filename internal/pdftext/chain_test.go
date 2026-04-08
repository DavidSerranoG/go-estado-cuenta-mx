package pdftext

import (
	"context"
	"errors"
	"testing"
)

type stubExtractor struct {
	name   string
	text   string
	err    error
	called *int
}

func (s stubExtractor) Name() string {
	return s.name
}

func (s stubExtractor) ExtractText(_ context.Context, _ []byte) (string, error) {
	if s.called != nil {
		*s.called = *s.called + 1
	}

	return s.text, s.err
}

func TestChainStopsOnFirstUsableText(t *testing.T) {
	t.Parallel()

	firstCalls := 0
	secondCalls := 0

	chain := NewChain(
		stubExtractor{name: "first", text: "  usable text  ", called: &firstCalls},
		stubExtractor{name: "second", text: "should not run", called: &secondCalls},
	)

	text, err := chain.ExtractText(context.Background(), []byte("pdf"))
	if err != nil {
		t.Fatalf("extract text: %v", err)
	}
	if text != "  usable text  " {
		t.Fatalf("unexpected text %q", text)
	}
	if firstCalls != 1 {
		t.Fatalf("expected first extractor to run once, got %d", firstCalls)
	}
	if secondCalls != 0 {
		t.Fatalf("expected second extractor to be skipped, got %d calls", secondCalls)
	}
}

func TestChainCollectsTypedAttempts(t *testing.T) {
	t.Parallel()

	chain := NewChain(
		stubExtractor{name: "go", err: newExtractorError("go", AttemptCodeFailed, errors.New("bad pdf"))},
		stubExtractor{name: "empty", text: "   "},
	)

	_, err := chain.ExtractText(context.Background(), []byte("pdf"))
	if err == nil {
		t.Fatal("expected extraction error")
	}

	var extractionErr *ExtractionError
	if !errors.As(err, &extractionErr) {
		t.Fatalf("expected ExtractionError, got %T", err)
	}
	if len(extractionErr.Attempts) != 2 {
		t.Fatalf("expected 2 attempts, got %d", len(extractionErr.Attempts))
	}
	if extractionErr.Attempts[0].Extractor != "go" || extractionErr.Attempts[0].Code != AttemptCodeFailed {
		t.Fatalf("unexpected first attempt %+v", extractionErr.Attempts[0])
	}
	if extractionErr.Attempts[1].Extractor != "empty" || extractionErr.Attempts[1].Code != AttemptCodeNoText {
		t.Fatalf("unexpected second attempt %+v", extractionErr.Attempts[1])
	}
}

func TestChainStopsAfterContextCancellation(t *testing.T) {
	t.Parallel()

	chain := NewChain(
		stubExtractor{name: "first", err: context.Canceled},
		stubExtractor{name: "second", text: "should not run"},
	)

	_, err := chain.ExtractText(context.Background(), []byte("pdf"))
	if err == nil {
		t.Fatal("expected extraction error")
	}

	var extractionErr *ExtractionError
	if !errors.As(err, &extractionErr) {
		t.Fatalf("expected ExtractionError, got %T", err)
	}
	if len(extractionErr.Attempts) != 1 {
		t.Fatalf("expected chain to stop after cancellation, got %d attempts", len(extractionErr.Attempts))
	}
	if !errors.Is(extractionErr.Attempts[0].Err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", extractionErr.Attempts[0].Err)
	}
}

func TestChainExtractCandidatesNormalizesEachSuccess(t *testing.T) {
	t.Parallel()

	chain := NewChain(
		stubExtractor{name: "first", text: "Pagina 1de6\nBBVA\n+\n1 221 00"},
		stubExtractor{name: "second", text: "HSBC\nNUMERO DE CUENTA"},
	)

	run, err := chain.ExtractCandidates(context.Background(), []byte("pdf"))
	if err != nil {
		t.Fatalf("extract candidates: %v", err)
	}
	if len(run.Candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(run.Candidates))
	}
	if len(run.Attempts) != 2 {
		t.Fatalf("expected 2 attempts, got %d", len(run.Attempts))
	}
	if run.Attempts[0].Code != AttemptCodeSucceeded || run.Attempts[1].Code != AttemptCodeSucceeded {
		t.Fatalf("unexpected attempts %+v", run.Attempts)
	}
	if run.Candidates[0].NormalizedText != "BBVA\n+$1221.00" {
		t.Fatalf("unexpected normalized text %q", run.Candidates[0].NormalizedText)
	}
	if run.Candidates[0].Quality.Score() <= 0 {
		t.Fatalf("expected positive quality score, got %+v", run.Candidates[0].Quality)
	}
}
