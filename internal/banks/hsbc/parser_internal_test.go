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
	if tx.Direction != "debit" {
		t.Fatalf("unexpected kind %q", tx.Direction)
	}
	if !tx.PostedAt.Equal(time.Date(2025, 4, 7, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected posted date %v", tx.PostedAt)
	}
}

func TestParseCardPeriodAcceptsOCRMonthDigits(t *testing.T) {
	t.Parallel()

	start, end, err := parseCardPeriod("a) Periodo:\n13-0ct-2025 al 12-Nov-2025")
	if err != nil {
		t.Fatalf("parse card period: %v", err)
	}
	if !start.Equal(time.Date(2025, 10, 13, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected start %v", start)
	}
	if !end.Equal(time.Date(2025, 11, 12, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected end %v", end)
	}
}

func TestParseFlexiblePeriodAcceptsSlashDates(t *testing.T) {
	t.Parallel()

	start, end, err := parseFlexiblePeriod("Período de 01/11/2025 al 30/11/2025")
	if err != nil {
		t.Fatalf("parse flexible period: %v", err)
	}
	if !start.Equal(time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected start %v", start)
	}
	if !end.Equal(time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected end %v", end)
	}
}

func TestHSBCDetectionScoreAcceptsFlexibleSlashPeriods(t *testing.T) {
	t.Parallel()

	text := `HSBC
CUENTA FLEXIBLE
Período de 01/11/2025 al 30/11/2025
DETALLE MOVIMIENTOS CUENTA FLEXIBLE No. 6529009644
18 AMI CUENTA HSBC`

	if score := hsbcDetectionScore(text); score <= 0 {
		t.Fatalf("expected positive HSBC score, got %d", score)
	}
}

func TestParseFlexibleInitialBalanceAcceptsOCRColumnReflow(t *testing.T) {
	t.Parallel()

	value, err := parseFlexibleInitialBalance(`RESUMEN DE CUENTAS
Saldo Inicial del
$ 7,595.73
FRACC REAL DEL MONTE
Periodo
Depósitos/
$ 25,000.00`)
	if err != nil {
		t.Fatalf("parse flexible initial balance: %v", err)
	}
	if value != 759573 {
		t.Fatalf("unexpected initial balance %d", value)
	}
}
