package statementpdf_test

import (
	"errors"
	"testing"

	"github.com/ledgermx/mxstatementpdf"
	"github.com/ledgermx/mxstatementpdf/hsbc"
)

func TestProcessorRejectsMissingParsers(t *testing.T) {
	t.Parallel()

	processor := statementpdf.New()

	_, err := processor.ParseText("HSBC MEXICO")
	if !errors.Is(err, statementpdf.ErrNoParsersConfigured) {
		t.Fatalf("expected ErrNoParsersConfigured, got %v", err)
	}
}

func TestProcessorDetectsHSBCText(t *testing.T) {
	t.Parallel()

	processor := statementpdf.New(
		statementpdf.WithParser(hsbc.New()),
	)

	statement, err := processor.ParseText(sampleHSBCText)
	if err != nil {
		t.Fatalf("parse text: %v", err)
	}

	if statement.Bank != "hsbc" {
		t.Fatalf("expected bank hsbc, got %q", statement.Bank)
	}
	if len(statement.Transactions) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(statement.Transactions))
	}
}

const sampleHSBCText = `HSBC MEXICO
NÚMERO DE CUENTA: 5470 7498 1184 6577
TU PAGO REQUERIDO ESTE PERIODO
15-Sep-2025 al 12-Oct-2025
16-Sep-202517-Sep-2025SU PAGO GRACIAS-         $25,000.00
12-Sep-202515-Sep-2025RUGR590104PR9 SERV GAS PREMIER       TIJ+$868.07`
