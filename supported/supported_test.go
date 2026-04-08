package supported_test

import (
	"context"
	"testing"

	edocuenta "github.com/DavidSerranoG/go-estado-cuenta-mx"
	"github.com/DavidSerranoG/go-estado-cuenta-mx/supported"
)

func TestParsersReturnsBuiltins(t *testing.T) {
	t.Parallel()

	parsers := supported.Parsers()
	if len(parsers) < 2 {
		t.Fatalf("expected at least 2 built-in parsers, got %d", len(parsers))
	}
}

func TestNewRegistersBuiltins(t *testing.T) {
	t.Parallel()

	processor := supported.New(
		edocuenta.WithExtractor(staticExtractor{text: sampleHSBCText}),
	)

	statement, err := processor.ParsePDF(context.Background(), []byte("pdf"))
	if err != nil {
		t.Fatalf("parse pdf with supported.New: %v", err)
	}
	if statement.Bank != edocuenta.BankHSBC {
		t.Fatalf("expected HSBC bank, got %q", statement.Bank)
	}
}

type staticExtractor struct {
	text string
}

func (e staticExtractor) ExtractText(context.Context, []byte) (string, error) {
	return e.text, nil
}

const sampleHSBCText = `HSBC MEXICO
NÚMERO DE CUENTA: 5470 7498 1184 6577
TU PAGO REQUERIDO ESTE PERIODO
15-Sep-2025 al 12-Oct-2025
16-Sep-202517-Sep-2025SU PAGO GRACIAS-         $25,000.00
12-Sep-202515-Sep-2025RUGR590104PR9 SERV GAS PREMIER       TIJ+$868.07`
