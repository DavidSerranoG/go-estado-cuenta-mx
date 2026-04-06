package hsbc_test

import (
	"testing"
	"time"

	"github.com/ledgermx/mxstatementpdf/hsbc"
)

func TestParseSyntheticStatement(t *testing.T) {
	t.Parallel()

	parser := hsbc.New()

	statement, err := parser.Parse(sampleText)
	if err != nil {
		t.Fatalf("parse statement: %v", err)
	}

	if statement.Bank != "hsbc" {
		t.Fatalf("expected hsbc, got %q", statement.Bank)
	}
	if statement.AccountNumber != "5470749811846577" {
		t.Fatalf("unexpected account %q", statement.AccountNumber)
	}
	if !statement.PeriodStart.Equal(time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected period start %v", statement.PeriodStart)
	}
	if !statement.PeriodEnd.Equal(time.Date(2025, 10, 12, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected period end %v", statement.PeriodEnd)
	}
	if len(statement.Transactions) != 3 {
		t.Fatalf("expected 3 transactions, got %d", len(statement.Transactions))
	}
	if statement.Transactions[0].Type != "abono" {
		t.Fatalf("unexpected first type %q", statement.Transactions[0].Type)
	}
	if statement.Transactions[1].AmountCents != 86807 {
		t.Fatalf("unexpected second amount %d", statement.Transactions[1].AmountCents)
	}
	if statement.Transactions[2].AmountCents != 19432 {
		t.Fatalf("unexpected third amount %d", statement.Transactions[2].AmountCents)
	}
}

func TestParseSyntheticFlexibleStatement(t *testing.T) {
	t.Parallel()

	parser := hsbc.New()

	statement, err := parser.Parse(flexibleSampleText)
	if err != nil {
		t.Fatalf("parse flexible statement: %v", err)
	}

	if statement.AccountNumber != "6529009644" {
		t.Fatalf("unexpected account %q", statement.AccountNumber)
	}
	if len(statement.Transactions) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(statement.Transactions))
	}
	if statement.Transactions[0].Type != "abono" {
		t.Fatalf("unexpected first type %q", statement.Transactions[0].Type)
	}
	if statement.Transactions[1].Type != "cargo" {
		t.Fatalf("unexpected second type %q", statement.Transactions[1].Type)
	}
	if statement.Transactions[1].Reference == "" {
		t.Fatalf("expected reference for second transaction")
	}
}

const sampleText = `HSBC 2Now Categoría: Oro
NÚMERO DE CUENTA: 5470 7498 1184 6577
TU PAGO REQUERIDO ESTE PERIODO
a) Periodo:
15-Sep-2025 al 12-Oct-2025
DESGLOSE DE MOVIMIENTOS
c) CARGOS, ABONOS Y COMPRAS REGULARES (NO A MESES)
Tarjeta titular 5470749811846577
i. Fecha de la operaciónii. Fecha de cargoiii. Descripción del movimientoiv. Monto
16-Sep-202517-Sep-2025SU PAGO GRACIAS-         $25,000.00
12-Sep-202515-Sep-2025RUGR590104PR9 SERV GAS PREMIER       TIJ+$868.07
02-Oct-202503-Oct-2025CLOUDFLARE             SAN FRANCISCO CA
MONEDA EXTRANJERA:                10.46 USD TC: 18.57743 DEL 02 DE OCTUBRE+$194.32`

const flexibleSampleText = `CUENTA FLEXIBLE
Estado de Cuenta
NÚMERO DE CUENTACLABE INTERBANCARIA
6529009644021028065290096448
4Período del01102025 al 31102025
4Saldo Inicial del
Periodo
$ 6,595.73
DETALLE MOVIMIENTOS CUENTA FLEXIBLE No.  6529009644
DíaDescripciónReferencia
SerialRetiroCargoDepósitoAbonoSaldo
20A MI HSBC                        081025008045221
201345
$ 36,000.00 $ 42,595.73
20PAGO DE TARJETA: 5470749811846577 EN BPI13655983
41234
$ 35,000.00 $ 7,595.73
Saldo Inicial $            6,595.73
Saldo Final $7,595.73`
