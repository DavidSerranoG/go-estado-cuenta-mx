package bbva

import (
	"testing"
	"time"

	edocuenta "github.com/DavidSerranoG/go-estado-cuenta-mx"
)

func TestClassifyByDescriptionResolvesKnownStatementPatterns(t *testing.T) {
	t.Parallel()

	cases := map[string]edocuenta.TransactionDirection{
		"ABONO POR CORRECCION WWW ALIEXPRESS COM":   edocuenta.TransactionDirectionCredit,
		"PAGO CUENTA DE TERCERO BNET 0105982138":    edocuenta.TransactionDirectionDebit,
		"ADYENMX*UBER EATS RFC: UPM 200220LK5":      edocuenta.TransactionDirectionDebit,
		"AMAZON MX MARKETPLACE RFC: ANE 140618P37":  edocuenta.TransactionDirectionDebit,
		"AMAZON MX RFC: ANE 140618P37":              edocuenta.TransactionDirectionDebit,
	}

	for description, want := range cases {
		if got := classifyByDescription(description); got != want {
			t.Fatalf("classifyByDescription(%q) = %q, want %q", description, got, want)
		}
	}
}

func TestParseRealTransactionsRepairsOCRLeadingODates(t *testing.T) {
	t.Parallel()

	text := `Estado de Cuenta
Libretón Básico
Periodo DEL 09/03/2019 AL 22/03/2019
No. de Cuenta 1528907610

Información Financiera MONEDA NACIONAL
Saldo Anterior 0.00
Depósitos / Abonos (+) 5 20,496.35
Retiros / Cargos (-) 2 5,087.20
Saldo Final 15,409.15

Detalle de Movimientos Realizados
O9/MAR 11/MAR APERTURA DE CUENTA
O9/MAR 11/MAR DEPOSITO EFECTIVO PRACTIC 5,500.00 5,500.00
MARO9 14:45 PRAC 7134 FOLIO:1237 Referencia ******7610
11/MAR 11/MAR DEPOSITO EFECTIVO PRACTIC 5,000.00 10,500.00 10,500.00
MAR11 19:25 PRAC 6849 FOLIO:2487 Referencia ******7610
18/MAR 19/MAR RECAUDACION DE IMPUE G 3,967.00 6,533.00 10,500.00
REF:02190EMP741923257217 CIE:0844985 Referencia UIA:4313111
19/MAR 19/MAR DEPOSITO EFECTIVO PRACTIC 5,500.00 12,033.00 10,912.80
MAR19 15:22 PRAC 4256 FOLIO:4605 Referencia ******7610
20/MAR 19/MAR ESTACION REFORMA * 1,120.20 10,912.80 10,912.80
RFC: ESP 080211R43 19:38 AUT: 248789 Referencia *****7339
21/MAR 21/MAR SPEI RECIBIDOBANAMEX 0 4,491.30 15,404.10 15,404.10
7910000PORTABILIDAD DE NOMINA Referencia 005324087 002
00005256782925183090
2019032140002NNNN0105797708092
DAVID ALBERTO,SERRANO/GARCIA
22/MAR 22/MAR 2242095435TRANSFERENCIA PR 5.05 15,409.15 15,409.15
Referencia OP

Total de Movimientos
TOTAL IMPORTE CARGOS 5,087.20 TOTAL MOVIMIENTOS CARGOS 2
TOTAL IMPORTE ABONOS 20,496.35 TOTAL MOVIMIENTOS ABONOS 5`

	periodStart := time.Date(2019, 3, 9, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2019, 3, 22, 0, 0, 0, 0, time.UTC)

	transactions, warnings := parseRealTransactions(text, periodStart, periodEnd)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if len(transactions) != 7 {
		t.Fatalf("len(transactions) = %d, want 7", len(transactions))
	}
	if transactions[0].PostedAt != time.Date(2019, 3, 9, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("first date = %v, want 2019-03-09", transactions[0].PostedAt)
	}
	if transactions[0].AmountCents != 550000 {
		t.Fatalf("first amount = %d, want 550000", transactions[0].AmountCents)
	}
	if transactions[1].AmountCents != 500000 {
		t.Fatalf("second amount = %d, want 500000", transactions[1].AmountCents)
	}
}

func TestParseRealTransactionsRejectsAdjacentOCRMoneyAsBalance(t *testing.T) {
	t.Parallel()

	text := `Estado de Cuenta
Libretón Dólares
Periodo DEL 01/07/2025 AL 31/07/2025
No. de Cuenta 0484984080

Información Financiera MONEDA DOLARES
Saldo Anterior 25,444.87
Depósitos / Abonos (+) 2 4,787.83
Retiros / Cargos (-) 3 20,052.00
Saldo Final 10,180.70

Detalle de Movimientos Realizados
02/JUL   02/JUL PAGO CUENTA DE TERCERO                                                                      2,289.83       27,734.70          27,734.70
                 0085065013 BNET 0111250892 Factura C4409
16/JUL   16/JUL TRASPASO ENTRE CUENTAS                                                         20,000.00
                 5463155.1002.01 FOLIO: 0000000 355059.80MXP
16/JUL   16/JUL PAGO CUENTA DE TERCERO                                                                      2,498.00       10,232.70          10,180.70
                 0047008005 BNET 0111250892 FACTURA B10D8
17/JUL   16/JUL SPO*QUARTYARD                                                                      22.00                   10,210.70          10,180.70
                 ******0434 USD 22.00TC001.0000AUT: 364009
18/JUL   16/JUL SQ *MEI SEMONES                                                                    30.00                   10,180.70          10,180.70
                 ******0434 USD 30.00TC001.0000AUT: 553261

Total de Movimientos
TOTAL IMPORTE CARGOS 20,052.00 TOTAL MOVIMIENTOS CARGOS 3
TOTAL IMPORTE ABONOS 4,787.83 TOTAL MOVIMIENTOS ABONOS 2`

	periodStart := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2025, 7, 31, 0, 0, 0, 0, time.UTC)

	transactions, warnings := parseRealTransactions(text, periodStart, periodEnd)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if len(transactions) != 5 {
		t.Fatalf("len(transactions) = %d, want 5", len(transactions))
	}
	if transactions[1].Description != "TRASPASO ENTRE CUENTAS" {
		t.Fatalf("second description = %q, want TRASPASO ENTRE CUENTAS", transactions[1].Description)
	}
	if transactions[1].AmountCents != 2000000 {
		t.Fatalf("second amount = %d, want 2000000", transactions[1].AmountCents)
	}
	if transactions[1].Direction != edocuenta.TransactionDirectionDebit {
		t.Fatalf("second direction = %q, want debit", transactions[1].Direction)
	}
	if transactions[2].AmountCents != 249800 {
		t.Fatalf("third amount = %d, want 249800", transactions[2].AmountCents)
	}
	if transactions[2].Direction != edocuenta.TransactionDirectionCredit {
		t.Fatalf("third direction = %q, want credit", transactions[2].Direction)
	}
}
