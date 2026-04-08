package hsbc

import (
	"testing"
	"time"
)

func TestExtractStandaloneOCRAmountAcceptsLeadingNoise(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		sign   string
		amount string
	}{
		"+1$453.85": {sign: "+", amount: "453.85"},
		"+4$129.90": {sign: "+", amount: "129.90"},
		"+|$822.45": {sign: "+", amount: "822.45"},
		"+]$214.00": {sign: "+", amount: "214.00"},
	}

	for input, want := range cases {
		input := input
		want := want

		t.Run(input, func(t *testing.T) {
			t.Parallel()

			sign, amount, ok := extractStandaloneOCRAmount(normalizeOCRCardLine(input))
			if !ok {
				t.Fatalf("expected %q to be detected as a standalone amount", input)
			}
			if sign != want.sign || amount != want.amount {
				t.Fatalf("unexpected parse for %q: sign=%q amount=%q", input, sign, amount)
			}
		})
	}
}

func TestParseSplitOCRCardTransactionAcceptsSplitSignAndAmountLines(t *testing.T) {
	t.Parallel()

	lines := []string{
		"04-Abr-2025",
		"",
		"07-Abr-2025",
		"",
		"MERCADOPAGO *BORDERPL Tijuana",
		"",
		"BCN",
		"",
		"+",
		"",
		"1,221.00",
	}

	periodStart := time.Date(2025, 3, 13, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2025, 4, 11, 0, 0, 0, 0, time.UTC)

	tx, endIdx, ok, err := parseSplitOCRCardTransaction(lines, 0, nil, periodStart, periodEnd)
	if !ok {
		t.Fatal("expected split OCR transaction to be detected")
	}
	if err != nil {
		t.Fatalf("expected split OCR transaction to parse, got %v", err)
	}
	if endIdx != 10 {
		t.Fatalf("expected parser to consume amount line, got endIdx=%d", endIdx)
	}
	if tx.Description != "MERCADOPAGO *BORDERPL Tijuana BCN" {
		t.Fatalf("unexpected description %q", tx.Description)
	}
	if tx.AmountCents != 122100 {
		t.Fatalf("unexpected amount %d", tx.AmountCents)
	}
	if tx.Kind != "debit" {
		t.Fatalf("unexpected kind %q", tx.Kind)
	}
	if !tx.PostedAt.Equal(time.Date(2025, 4, 7, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected posted date %v", tx.PostedAt)
	}
}
