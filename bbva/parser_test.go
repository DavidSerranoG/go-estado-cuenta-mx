package bbva_test

import (
	"testing"
	"time"

	"github.com/ledgermx/mxstatementpdf/bbva"
)

func TestParseSyntheticStatement(t *testing.T) {
	t.Parallel()

	parser := bbva.New()

	statement, err := parser.Parse(sampleText)
	if err != nil {
		t.Fatalf("parse statement: %v", err)
	}

	if statement.Bank != "bbva" {
		t.Fatalf("expected bbva, got %q", statement.Bank)
	}
	if statement.AccountNumber != "1234567890" {
		t.Fatalf("unexpected account %q", statement.AccountNumber)
	}
	if !statement.PeriodStart.Equal(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected period start %v", statement.PeriodStart)
	}
	if !statement.PeriodEnd.Equal(time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected period end %v", statement.PeriodEnd)
	}
	if len(statement.Transactions) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(statement.Transactions))
	}
	if statement.Transactions[0].Type != "abono" {
		t.Fatalf("unexpected first type %q", statement.Transactions[0].Type)
	}
	if statement.Transactions[1].AmountCents != 250000 {
		t.Fatalf("unexpected second amount %d", statement.Transactions[1].AmountCents)
	}
}

const sampleText = `BBVA MEXICO
Cuenta: 1234567890
Periodo: 01/03/2026 - 31/03/2026
01/03/2026 NOMINA MARZO ABONO 15000.00 15000.00
02/03/2026 PAGO TARJETA CARGO 2500.00 12500.00`
