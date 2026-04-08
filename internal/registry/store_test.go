package registry_test

import (
	"testing"

	"github.com/DavidSerranoG/go-estado-cuenta-mx/internal/registry"
)

type parserStub struct {
	bank  string
	match string
	score int
}

func (p parserStub) Bank() string {
	return p.bank
}

func (p parserStub) CanParse(text string) bool {
	return p.match != "" && text == p.match
}

func (p parserStub) DetectionScore(string) int {
	return p.score
}

type plainParserStub struct {
	bank  string
	match string
}

func (p plainParserStub) Bank() string {
	return p.bank
}

func (p plainParserStub) CanParse(text string) bool {
	return p.match != "" && text == p.match
}

func TestStorePrefersHighestDetectionScore(t *testing.T) {
	t.Parallel()

	store := registry.New[parserStub](
		parserStub{bank: "first", score: 3},
		parserStub{bank: "second", score: 9},
	)

	got, ok := store.FindByText("statement")
	if !ok {
		t.Fatal("expected parser match")
	}
	if got.Bank() != "second" {
		t.Fatalf("expected highest-score parser, got %q", got.Bank())
	}
}

func TestStoreFallsBackToCanParseForLegacyParsers(t *testing.T) {
	t.Parallel()

	store := registry.New[plainParserStub](
		plainParserStub{bank: "legacy", match: "ok"},
	)

	got, ok := store.FindByText("ok")
	if !ok {
		t.Fatal("expected parser match")
	}
	if got.Bank() != "legacy" {
		t.Fatalf("unexpected bank %q", got.Bank())
	}
}

func TestStoreFindByBankNormalizesCaseAndWhitespace(t *testing.T) {
	t.Parallel()

	store := registry.New[plainParserStub](
		plainParserStub{bank: " HSBC "},
	)

	got, ok := store.FindByBank(" hsbc ")
	if !ok {
		t.Fatal("expected parser lookup by bank")
	}
	if got.Bank() != " HSBC " {
		t.Fatalf("unexpected bank %q", got.Bank())
	}
}
