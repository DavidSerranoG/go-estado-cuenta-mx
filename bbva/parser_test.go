package bbva_test

import (
	"strings"
	"testing"
	"time"

	"github.com/DavidSerranoG/go-estado-cuenta-mx/bbva"
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
	if statement.Transactions[0].Kind != "credit" {
		t.Fatalf("unexpected first kind %q", statement.Transactions[0].Kind)
	}
	if statement.Transactions[1].AmountCents != 250000 {
		t.Fatalf("unexpected second amount %d", statement.Transactions[1].AmountCents)
	}
}

func TestParseRealStyleStatement(t *testing.T) {
	t.Parallel()

	parser := bbva.New()

	statement, err := parser.Parse(realStyleText)
	if err != nil {
		t.Fatalf("parse real style statement: %v", err)
	}

	if statement.AccountNumber != "1528907610" {
		t.Fatalf("unexpected account %q", statement.AccountNumber)
	}
	if statement.Currency != "MXN" {
		t.Fatalf("unexpected currency %q", statement.Currency)
	}
	if !statement.PeriodStart.Equal(time.Date(2025, 12, 23, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected period start %v", statement.PeriodStart)
	}
	if !statement.PeriodEnd.Equal(time.Date(2026, 1, 22, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected period end %v", statement.PeriodEnd)
	}
	if len(statement.Transactions) != 3 {
		t.Fatalf("expected 3 transactions, got %d", len(statement.Transactions))
	}
	if !statement.Transactions[0].PostedAt.Equal(time.Date(2025, 12, 26, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected first posted date %v", statement.Transactions[0].PostedAt)
	}
	if !statement.Transactions[2].PostedAt.Equal(time.Date(2026, 1, 7, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected third posted date %v", statement.Transactions[2].PostedAt)
	}
	for i, tx := range statement.Transactions {
		if tx.Kind != "debit" {
			t.Fatalf("transaction %d expected debit, got %q", i, tx.Kind)
		}
	}
	if statement.Transactions[0].BalanceCents == nil || *statement.Transactions[0].BalanceCents != 4844196 {
		t.Fatalf("unexpected first balance %+v", statement.Transactions[0].BalanceCents)
	}
}

func TestParseRealStyleStatementInfersMissingBalanceFromSummary(t *testing.T) {
	t.Parallel()

	parser := bbva.New()

	statement, err := parser.Parse(realStyleMissingBalanceText)
	if err != nil {
		t.Fatalf("parse statement with inferred balance: %v", err)
	}

	if len(statement.Transactions) != 7 {
		t.Fatalf("expected 7 transactions, got %d", len(statement.Transactions))
	}
	if statement.Transactions[3].Kind != "credit" {
		t.Fatalf("expected inferred transfer to be credit, got %q", statement.Transactions[3].Kind)
	}
	if statement.Transactions[3].BalanceCents == nil || *statement.Transactions[3].BalanceCents != 35919460 {
		t.Fatalf("unexpected inferred balance %+v", statement.Transactions[3].BalanceCents)
	}
	if statement.Transactions[4].Kind != "debit" {
		t.Fatalf("expected following transfer to be debit, got %q", statement.Transactions[4].Kind)
	}
}

func TestParseRealStyleStatementRepairsAmountUsingRunningBalance(t *testing.T) {
	t.Parallel()

	parser := bbva.New()

	statement, err := parser.Parse(realStyleUSDText)
	if err != nil {
		t.Fatalf("parse usd statement: %v", err)
	}

	if statement.Currency != "USD" {
		t.Fatalf("unexpected currency %q", statement.Currency)
	}
	if len(statement.Transactions) != 3 {
		t.Fatalf("expected 3 transactions, got %d", len(statement.Transactions))
	}
	if statement.Transactions[0].Kind != "credit" {
		t.Fatalf("unexpected first kind %q", statement.Transactions[0].Kind)
	}
	if statement.Transactions[1].Kind != "debit" {
		t.Fatalf("unexpected second kind %q", statement.Transactions[1].Kind)
	}
	if statement.Transactions[1].AmountCents != 36830 {
		t.Fatalf("unexpected repaired amount %d", statement.Transactions[1].AmountCents)
	}
	if statement.Transactions[2].Kind != "credit" {
		t.Fatalf("unexpected third kind %q", statement.Transactions[2].Kind)
	}
}

func TestParseStatementFallsBackToCLABEAndSpacedCompactDates(t *testing.T) {
	t.Parallel()

	parser := bbva.New()

	statement, err := parser.Parse(realStyleCLABEFallbackText)
	if err != nil {
		t.Fatalf("parse clabe fallback statement: %v", err)
	}

	if statement.AccountNumber != "0484984080" {
		t.Fatalf("unexpected account %q", statement.AccountNumber)
	}
	if statement.Currency != "USD" {
		t.Fatalf("unexpected currency %q", statement.Currency)
	}
	if !statement.PeriodStart.Equal(time.Date(2026, 2, 23, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected period start %v", statement.PeriodStart)
	}
	if !statement.PeriodEnd.Equal(time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected period end %v", statement.PeriodEnd)
	}
	if len(statement.Transactions) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(statement.Transactions))
	}
	if statement.Transactions[0].Kind != "credit" {
		t.Fatalf("unexpected first kind %q", statement.Transactions[0].Kind)
	}
	if statement.Transactions[1].Kind != "debit" {
		t.Fatalf("unexpected second kind %q", statement.Transactions[1].Kind)
	}
	if statement.Transactions[1].BalanceCents == nil || *statement.Transactions[1].BalanceCents != 100433 {
		t.Fatalf("unexpected second balance %+v", statement.Transactions[1].BalanceCents)
	}
}

func TestParseCreditCardStatement(t *testing.T) {
	t.Parallel()

	parser := bbva.New()

	statement, err := parser.Parse(cardStatementText)
	if err != nil {
		t.Fatalf("parse credit card statement: %v", err)
	}

	if statement.AccountNumber != "XXXXXX9919" {
		t.Fatalf("unexpected account %q", statement.AccountNumber)
	}
	if statement.Currency != "MXN" {
		t.Fatalf("unexpected currency %q", statement.Currency)
	}
	if !statement.PeriodStart.Equal(time.Date(2026, 2, 25, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected period start %v", statement.PeriodStart)
	}
	if !statement.PeriodEnd.Equal(time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected period end %v", statement.PeriodEnd)
	}
	if len(statement.Transactions) != 3 {
		t.Fatalf("expected 3 transactions, got %d", len(statement.Transactions))
	}
	if statement.Transactions[0].Kind != "credit" {
		t.Fatalf("unexpected first kind %q", statement.Transactions[0].Kind)
	}
	if statement.Transactions[1].Kind != "debit" {
		t.Fatalf("unexpected second kind %q", statement.Transactions[1].Kind)
	}
	if statement.Transactions[2].AmountCents != 19456 {
		t.Fatalf("unexpected third amount %d", statement.Transactions[2].AmountCents)
	}
	if !strings.Contains(statement.Transactions[2].Description, "TIPO DE CAMBIO") {
		t.Fatalf("expected continuation to be joined, got %q", statement.Transactions[2].Description)
	}
}

func TestParseCreditCardStatementWithOCRNoise(t *testing.T) {
	t.Parallel()

	parser := bbva.New()

	statement, err := parser.Parse(cardStatementOCRNoiseText)
	if err != nil {
		t.Fatalf("parse noisy credit card statement: %v", err)
	}

	if len(statement.Transactions) != 3 {
		t.Fatalf("expected 3 transactions, got %d", len(statement.Transactions))
	}
	if statement.Transactions[0].AmountCents != 1529764 {
		t.Fatalf("unexpected first amount %d", statement.Transactions[0].AmountCents)
	}
	if statement.Transactions[1].PostedAt != time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("unexpected second posted date %v", statement.Transactions[1].PostedAt)
	}
}

func TestParseCreditCardStatementRejectsPartialTransactionsWhenTotalsMismatch(t *testing.T) {
	t.Parallel()

	parser := bbva.New()

	_, err := parser.Parse(cardStatementPartialText)
	if err == nil {
		t.Fatal("expected credit card parse error")
	}
	if !strings.Contains(err.Error(), "credit card text incomplete") {
		t.Fatalf("unexpected error %v", err)
	}
}

const sampleText = `BBVA MEXICO
Cuenta: 1234567890
Periodo: 01/03/2026 - 31/03/2026
01/03/2026 NOMINA MARZO ABONO 15000.00 15000.00
02/03/2026 PAGO TARJETA CARGO 2500.00 12500.00`

const realStyleText = `BBVA
PeriodoDEL 23/12/2025 AL 22/01/2026Fecha de Corte 22/01/2026No. de Cuenta1528907610No. de ClienteC0000000No. Cuenta CLABE012 028 01528907610 8
Información FinancieraMONEDA NACIONALLibretón Básico Cuenta Digital
ComportamientoSaldo Anterior63,311.83Depósitos / Abonos (+)0.00Retiros / Cargos (-)16,719.87Saldo Final46,591.96
Detalle de Movimientos RealizadosFECHASALDOOPERLIQDESCRIPCIONREFERENCIACARGOSABONOSOPERACIONLIQUIDACION26/DIC26/DICPAGO TARJETA DE CREDITO14,869.8748,441.9648,441.96 CUENTA: BMOV Referencia 575649607129/DIC29/DICSPEI ENVIADO BANORTE1,000.0047,441.9647,441.96 0912250consulta Referencia 009312814207/ENE07/ENEPAGO CUENTA DE TERCERO850.0046,591.9646,591.96 BNET 0489009205 Transf Referencia 0097399462TOTAL IMPORTE CARGOS16,719.87TOTAL MOVIMIENTOS CARGOS3TOTAL IMPORTE ABONOS0.00TOTAL MOVIMIENTOS ABONOS0`

const realStyleMissingBalanceText = `BBVA
PeriodoDEL 23/01/2026 AL 22/02/2026Fecha de Corte 22/02/2026No. de Cuenta1528907610No. de ClienteC0000000No. Cuenta CLABE012 028 01528907610 8
Información FinancieraMONEDA NACIONALLibretón Básico Cuenta Digital
ComportamientoSaldo Anterior19,192.96Depósitos / Abonos (+)379,090.20Retiros / Cargos (-)391,791.56Saldo Final6,491.60
Detalle de Movimientos RealizadosFECHASALDOOPERLIQDESCRIPCIONREFERENCIACARGOSABONOSOPERACIONLIQUIDACION27/ENE27/ENEPAGO TARJETA DE CREDITO14,088.565,104.405,104.40 CUENTA: BMOV Referencia 853209811029/ENE29/ENESPEI RECIBIDOSCOTIABANK52,200.0057,304.4057,304.40 Referencia 015781111804/FEB05/FEBSPEI ENVIADO HSBC25,000.0032,304.4057,304.40 Referencia 007951182112/FEB12/FEBTRASPASO ENTRE CUENTAS326,890.20 FOLIO: 0000000 20000.00USD Referencia 8304315.1002.0112/FEB12/FEBSPEI ENVIADO HSBC330,000.0029,194.6029,194.60 Referencia 006658335216/FEB16/FEBSPEI ENVIADO HSBC14,000.0015,194.6015,194.60 Referencia 008948902218/FEB18/FEBSAT8,703.006,491.606,491.60 REF:04261TEST950048844224TOTAL IMPORTE CARGOS391,791.56TOTAL MOVIMIENTOS CARGOS5TOTAL IMPORTE ABONOS379,090.20TOTAL MOVIMIENTOS ABONOS2`

const realStyleUSDText = `BBVA
PeriodoDEL 01/01/2026 AL 31/01/2026Fecha de Corte 31/01/2026No. de Cuenta0484984080No. de ClienteC0000000No. Cuenta CLABE012028004849840808
Información FinancieraMONEDA DOLARESLibretón Dólares
ComportamientoSaldo Anterior24,168.62Depósitos / Abonos (+)4,787.83Retiros / Cargos (-)368.30Saldo Final28,588.15
Detalle de Movimientos RealizadosFECHASALDOOPERLIQDESCRIPCIÓNREFERENCIACARGOSABONOSOPERACIÓNLIQUIDACIÓN02/ENE02/ENEPAGO CUENTA DE TERCERO2,498.0026,666.6226,666.620013028016 BNET 0111250892 Factura A2AE205/ENE04/ENEWAL-MART #3947368.3026,298.3226,298.32******0434 USD 368.30TC001.0000AUT: 87569816/ENE16/ENEPAGO CUENTA DE TERCERO2,289.8328,588.1528,588.150021007026 BNET 0111250892 Factura CB2FATOTAL IMPORTE CARGOS368.30TOTAL MOVIMIENTOS CARGOS1TOTAL IMPORTE ABONOS4,787.83TOTAL MOVIMIENTOS ABONOS2`

const realStyleCLABEFallbackText = `BBVA
PeriodoDEL23/02/2026AL24/03/2026Fecha de Corte24/03/2026No. Cuenta CLABE012 028 00484984080 8
Información FinancieraMONEDA DÓLARESLibretón Dólares
ComportamientoSaldo Anterior1,000.33Depósitos / Abonos (+)500.00Retiros / Cargos (-)496.00Saldo Final1,004.33
Detalle de Movimientos RealizadosFECHASALDOOPERLIQDESCRIPCIONREFERENCIACARGOSABONOSOPERACIONLIQUIDACION24/FEB 24/FEB DEPOSITO EN EFECTIVO500.001,500.331,500.33 SUCURSAL 00012705/MAR 05/MAR PAGO TARJETA496.001,004.331,004.33 REFERENCIA 0093128142TOTAL IMPORTE CARGOS496.00TOTAL MOVIMIENTOS CARGOS1TOTAL IMPORTE ABONOS500.00TOTAL MOVIMIENTOS ABONOS1`

const cardStatementText = `BBVA
TARJETA AZUL BBVA (CLASICA)
Número de tarjeta: 4772913064069919
TU PAGO REQUERIDO ESTE PERIODO
Periodo: 25-feb-2026 al 24-mar-2026
PAGO PARA NO GENERAR INTERESES $413.24
DESGLOSE DE MOVIMIENTOS
CARGOS,COMPRAS Y ABONOS REGULARES(NO A MESES) Tarjeta titular: XXXXXXXXXXXX9919
Fecha de la operación Fecha de cargo Descripción del movimiento Monto
04-mar-2026 04-mar-2026 BMOVIL.PAGO TDC - $15,297.64
Número de cuenta: XXXXXX9919
CARGOS,COMPRAS Y ABONOS REGULARES(NO A MESES) Tarjeta titular: XXXXXXXXXXXX9919
04-mar-2026 06-mar-2026 AMAZON WEB SERVICES ; Tarjeta Digital ***2932 + $218.68
15-mar-2026 17-mar-2026 MVBILLING.COM ; Tarjeta Digital ***2932 + $194.56
MXP $194.56 TIPO DE CAMBIO $1.00
TOTAL CARGOS $413.24
TOTAL ABONOS -$15,297.64
ATENCION DE QUEJAS`

const cardStatementOCRNoiseText = `Pagina 1de6
DAVID ALBERTO SERRANO GARCIA TU PAGO REQUERIDO ESTE PERIODO
TARJETA AZUL BBVA (CLASICA)
Numero de tarjeta: 4772913064069919
Periodo: 25-feb-2026 al 24-mar-2026
Pago para no generar intereses: $413.24
Numero de cuenta: XXXXXX9919 Pagina 2de6
DESGLOSE DE MOVIMIENTOS.
CARGOS,COMPRAS Y ABONOS REGULARES(NO A MESES) Tarjeta titular: XXXXXXXXXXXX9919
Fecha Fecha
dela de cargo Descripcién del movimiento Monto
operacién
04-mar-2026 04-mar-2026 BMOVIL.PAGO TDC. - $15,297.64
Numero de cuenta: XXXXXX9919 Pagina 3de6
CARGOS,COMPRAS Y ABONOS REGULARES(NO A MESES) Tarjeta titular: XXXXXXXXXXXX9919
Fecha Fecha
dela de cargo Descripcién del movimiento Monto
operacién
IVA :$ 0.00 Interes: $ 0.00 Comisiones:$0.00 Capital:$15,297.64 Capital
de promocion:$0.00 Pago excedente:$0.00
04-mar-2026 06-mar-2026 AMAZON WEB SERVICES ; Tarjeta Digital ***2932 + $218.68
15-mar-2026 17-mar-2026 MVBILLING.COM ; Tarjeta Digital ***2932 + $194.56
MXP $194.56 TIPO DE CAMBIO $1.00
TOTAL CARGOS)| $413.24
TOTAL ABONOS -$15,297.64
ATENCION DE QUEJAS`

const cardStatementPartialText = `BBVA
TARJETA AZUL BBVA (CLASICA)
Número de tarjeta: 4772913064069919
Periodo: 25-feb-2026 al 24-mar-2026
DESGLOSE DE MOVIMIENTOS
CARGOS,COMPRAS Y ABONOS REGULARES(NO A MESES) Tarjeta titular: XXXXXXXXXXXX9919
04-mar-2026 04-mar-2026 BMOVIL.PAGO TDC - $15,297.64
Número de cuenta: XXXXXX9919
04-mar-2026 06-mar-2026 AMAZON WEB SERVICES ; Tarjeta Digital ***2932 + $218.68
TOTAL CARGOS $413.24
TOTAL ABONOS -$15,297.64`
