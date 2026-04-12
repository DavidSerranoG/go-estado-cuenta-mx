package edocuenta_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	edocuenta "github.com/DavidSerranoG/go-estado-cuenta-mx"
	"github.com/DavidSerranoG/go-estado-cuenta-mx/internal/pdftext"
)

func TestProcessorRescueCandidateCanBeatPrimaryOnTransactionCount(t *testing.T) {
	t.Parallel()

	processor := edocuenta.New(
		edocuenta.WithExtractor(candidateExtractor{
			name:  "primary",
			texts: []string{"FAKE BANK HIGH SCORE ONE TX"},
		}),
		edocuenta.WithRescueExtractor(candidateExtractor{
			name:  "rescue",
			texts: []string{"FAKE BANK LOW SCORE TWO TX"},
		}),
		edocuenta.WithParser(fakeCandidateParser{}),
	)

	result, err := processor.ParsePDFResult(context.Background(), []byte("pdf"))
	if err != nil {
		t.Fatalf("parse pdf result: %v", err)
	}
	if len(result.Statement.Transactions) != 2 {
		t.Fatalf("expected rescue candidate to win with 2 transactions, got %d", len(result.Statement.Transactions))
	}
	if result.Extraction.SelectedExtractor != "rescue" || !result.Extraction.UsedRescue {
		t.Fatalf("unexpected extraction diagnostics %+v", result.Extraction)
	}
}

func TestProcessorPrimaryCandidateWinsTieOnDetectionScore(t *testing.T) {
	t.Parallel()

	processor := edocuenta.New(
		edocuenta.WithExtractor(candidateExtractor{
			name:  "primary",
			texts: []string{"FAKE BANK HIGH SCORE ONE TX"},
		}),
		edocuenta.WithRescueExtractor(candidateExtractor{
			name:  "rescue",
			texts: []string{"FAKE BANK LOW SCORE ONE TX"},
		}),
		edocuenta.WithParser(fakeCandidateParser{}),
	)

	result, err := processor.ParsePDFResult(context.Background(), []byte("pdf"))
	if err != nil {
		t.Fatalf("parse pdf result: %v", err)
	}
	if result.Extraction.SelectedExtractor != "primary" || result.Extraction.UsedRescue {
		t.Fatalf("unexpected extraction diagnostics %+v", result.Extraction)
	}
}

func TestProcessorPrefersHigherConfidenceCandidateOverMoreTransactions(t *testing.T) {
	t.Parallel()

	processor := edocuenta.New(
		edocuenta.WithExtractor(candidateExtractor{
			name:  "primary",
			texts: []string{"FAKE BANK HIGH SCORE ONE TX"},
		}),
		edocuenta.WithRescueExtractor(candidateExtractor{
			name:  "rescue",
			texts: []string{"FAKE BANK LOW SCORE TWO TX MISSING ACCOUNT"},
		}),
		edocuenta.WithParser(fakeCandidateParser{}),
	)

	result, err := processor.ParsePDFResult(context.Background(), []byte("pdf"))
	if err != nil {
		t.Fatalf("parse pdf result: %v", err)
	}
	if result.Extraction.SelectedExtractor != "primary" {
		t.Fatalf("expected primary candidate to win on confidence, got %q", result.Extraction.SelectedExtractor)
	}
	if result.Diagnostics.Confidence != edocuenta.ParseConfidenceHigh {
		t.Fatalf("expected high confidence result, got %q", result.Diagnostics.Confidence)
	}
}

type candidateExtractor struct {
	name  string
	texts []string
}

func (e candidateExtractor) Name() string {
	return e.name
}

func (e candidateExtractor) ExtractText(context.Context, []byte) (string, error) {
	if len(e.texts) == 0 {
		return "", nil
	}
	return e.texts[0], nil
}

func (e candidateExtractor) ExtractCandidates(context.Context, []byte) (pdftext.CandidateRun, error) {
	run := pdftext.CandidateRun{
		Candidates: make([]pdftext.Candidate, 0, len(e.texts)),
		Attempts:   make([]pdftext.Attempt, 0, len(e.texts)),
	}

	for _, text := range e.texts {
		run.Candidates = append(run.Candidates, pdftext.NewCandidate(e.name, text))
		run.Attempts = append(run.Attempts, pdftext.Attempt{
			Extractor: e.name,
			Code:      pdftext.AttemptCodeSucceeded,
		})
	}

	return run, nil
}

type fakeCandidateParser struct{}

func (fakeCandidateParser) Bank() string {
	return "fake"
}

func (fakeCandidateParser) CanParse(text string) bool {
	return strings.Contains(text, "FAKE BANK")
}

func (fakeCandidateParser) DetectionScore(text string) int {
	score := 0
	if strings.Contains(text, "FAKE BANK") {
		score += 5
	}
	if strings.Contains(text, "HIGH SCORE") {
		score += 10
	}
	if strings.Contains(text, "LOW SCORE") {
		score += 1
	}
	return score
}

func (p fakeCandidateParser) Parse(text string) (edocuenta.Statement, error) {
	result, err := p.ParseResult(text)
	if err != nil {
		return edocuenta.Statement{}, err
	}
	return result.Statement, nil
}

func (fakeCandidateParser) ParseResult(text string) (edocuenta.ParseResult, error) {
	if !strings.Contains(text, "FAKE BANK") {
		return edocuenta.ParseResult{}, fmt.Errorf("fake: unsupported text")
	}

	count := 1
	if strings.Contains(text, "TWO TX") {
		count = 2
	}

	transactions := make([]edocuenta.Transaction, 0, count)
	for i := 0; i < count; i++ {
		transactions = append(transactions, edocuenta.Transaction{
			PostedAt:    time.Date(2026, 4, i+1, 0, 0, 0, 0, time.UTC),
			Description: fmt.Sprintf("tx-%d", i+1),
			Direction:   edocuenta.TransactionDirectionDebit,
			AmountCents: int64(100 + i),
		})
	}

	return edocuenta.ParseResult{
		Statement: edocuenta.Statement{
			Bank:          "fake",
			AccountNumber: fakeAccountNumber(text),
			Currency:      edocuenta.CurrencyMXN,
			PeriodStart:   time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			PeriodEnd:     time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC),
			Transactions:  transactions,
		},
	}, nil
}

func fakeAccountNumber(text string) string {
	if strings.Contains(text, "MISSING ACCOUNT") {
		return ""
	}
	return "1234"
}
