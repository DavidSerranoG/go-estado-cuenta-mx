package hsbc_test

import (
	"testing"
	"time"

	"github.com/DavidSerranoG/go-estado-cuenta-mx/hsbc"
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
	if statement.Transactions[0].Kind != "credit" {
		t.Fatalf("unexpected first kind %q", statement.Transactions[0].Kind)
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
	if statement.Transactions[0].Kind != "credit" {
		t.Fatalf("unexpected first kind %q", statement.Transactions[0].Kind)
	}
	if statement.Transactions[1].Kind != "debit" {
		t.Fatalf("unexpected second kind %q", statement.Transactions[1].Kind)
	}
	if statement.Transactions[1].Reference == "" {
		t.Fatalf("expected reference for second transaction")
	}
}

func TestParseFlexibleStatementStopsBeforeAppendixSections(t *testing.T) {
	t.Parallel()

	parser := hsbc.New()

	result, err := parser.ParseResult(flexibleWithAppendixText)
	if err != nil {
		t.Fatalf("parse flexible statement with appendix: %v", err)
	}
	statement := result.Statement

	if len(statement.Transactions) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(statement.Transactions))
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Warnings)
	}
	if statement.Transactions[0].Kind != "credit" {
		t.Fatalf("unexpected first kind %q", statement.Transactions[0].Kind)
	}
	if statement.Transactions[1].Kind != "debit" {
		t.Fatalf("unexpected second kind %q", statement.Transactions[1].Kind)
	}
}

func TestParseFlexibleStatementAcceptsNumericReferenceHeaders(t *testing.T) {
	t.Parallel()

	parser := hsbc.New()

	result, err := parser.ParseResult(flexibleNumericHeaderText)
	if err != nil {
		t.Fatalf("parse flexible statement with numeric header: %v", err)
	}
	statement := result.Statement

	if len(statement.Transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(statement.Transactions))
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Warnings)
	}
	if statement.Transactions[0].Kind != "credit" {
		t.Fatalf("unexpected kind %q", statement.Transactions[0].Kind)
	}
	if statement.Transactions[0].Reference == "" {
		t.Fatalf("expected reference to be preserved")
	}
}

func TestParseOCRLikeFlexibleStatement(t *testing.T) {
	t.Parallel()

	parser := hsbc.New()

	result, err := parser.ParseResult(ocrLikeFlexibleText)
	if err != nil {
		t.Fatalf("parse ocr-like flexible statement: %v", err)
	}
	statement := result.Statement

	if len(statement.Transactions) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(statement.Transactions))
	}
	if statement.Transactions[0].Kind != "credit" {
		t.Fatalf("unexpected first kind %q", statement.Transactions[0].Kind)
	}
	if statement.Transactions[0].AmountCents != 2500000 {
		t.Fatalf("unexpected first amount %d", statement.Transactions[0].AmountCents)
	}
	if statement.Transactions[1].Kind != "debit" {
		t.Fatalf("unexpected second kind %q", statement.Transactions[1].Kind)
	}
	if statement.Transactions[1].AmountCents != 1935542 {
		t.Fatalf("unexpected second amount %d", statement.Transactions[1].AmountCents)
	}
	if statement.Transactions[1].Reference == "" {
		t.Fatalf("expected flexible OCR reference to be preserved")
	}
}

func TestParseOCRLikeCardStatement(t *testing.T) {
	t.Parallel()

	parser := hsbc.New()

	statement, err := parser.Parse(ocrLikeCardText)
	if err != nil {
		t.Fatalf("parse ocr-like card statement: %v", err)
	}

	if len(statement.Transactions) != 6 {
		t.Fatalf("expected 6 transactions, got %d", len(statement.Transactions))
	}
	if statement.Transactions[0].Kind != "credit" {
		t.Fatalf("unexpected first kind %q", statement.Transactions[0].Kind)
	}
	if statement.Transactions[0].AmountCents != 2065295 {
		t.Fatalf("unexpected payment amount %d", statement.Transactions[0].AmountCents)
	}
	if statement.Transactions[1].AmountCents != 58600 {
		t.Fatalf("unexpected ramen amount %d", statement.Transactions[1].AmountCents)
	}

	var foreignFound bool
	for _, tx := range statement.Transactions {
		if tx.Description != "OPENAI *CHATGPT SUBSCR SAN FRANCISCO CA MONEDA EXTRANJERA: 20.00 USD TC: 18.0705 DEL 08 DE ENERO" {
			continue
		}
		foreignFound = true
		if tx.AmountCents != 36141 {
			t.Fatalf("unexpected foreign amount %d", tx.AmountCents)
		}
		if tx.Kind != "debit" {
			t.Fatalf("unexpected foreign kind %q", tx.Kind)
		}
	}
	if !foreignFound {
		t.Fatal("expected foreign currency transaction to be parsed")
	}
	if !statement.Transactions[5].PostedAt.Equal(time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected final posted date %v", statement.Transactions[5].PostedAt)
	}
}

func TestParsePSM11LikeCardStatement(t *testing.T) {
	t.Parallel()

	parser := hsbc.New()

	statement, err := parser.Parse(psm11LikeCardText)
	if err != nil {
		t.Fatalf("parse psm11-like card statement: %v", err)
	}

	if len(statement.Transactions) != 6 {
		t.Fatalf("expected 6 transactions, got %d", len(statement.Transactions))
	}
	if statement.Transactions[0].Kind != "credit" {
		t.Fatalf("unexpected first kind %q", statement.Transactions[0].Kind)
	}

	var foreignFound bool
	for _, tx := range statement.Transactions {
		if tx.Description != "OPENAI *CHATGPT SUBSCR SAN FRANCISCO CA MONEDA EXTRANJERA: 20.00 USD TC: 18.0705 DEL 08 DE ENERO" {
			continue
		}
		foreignFound = true
		if tx.AmountCents != 36141 {
			t.Fatalf("unexpected foreign amount %d", tx.AmountCents)
		}
	}
	if !foreignFound {
		t.Fatal("expected foreign transaction in psm11 sample")
	}
}

func TestParsePSM11LikeCardStatementWithOCRAmountArtifacts(t *testing.T) {
	t.Parallel()

	parser := hsbc.New()

	result, err := parser.ParseResult(psm11ArtifactCardText)
	if err != nil {
		t.Fatalf("parse artifact-heavy psm11 statement: %v", err)
	}
	statement := result.Statement

	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Warnings)
	}
	if len(statement.Transactions) != 4 {
		t.Fatalf("expected 4 transactions, got %d", len(statement.Transactions))
	}
	if statement.Transactions[0].AmountCents != 12990 {
		t.Fatalf("unexpected first amount %d", statement.Transactions[0].AmountCents)
	}
	if statement.Transactions[1].AmountCents != 122100 {
		t.Fatalf("unexpected second amount %d", statement.Transactions[1].AmountCents)
	}
	if statement.Transactions[2].AmountCents != 28490 {
		t.Fatalf("unexpected third amount %d", statement.Transactions[2].AmountCents)
	}
	if statement.Transactions[3].AmountCents != 41580 {
		t.Fatalf("unexpected foreign amount %d", statement.Transactions[3].AmountCents)
	}
	if statement.Transactions[3].Kind != "debit" {
		t.Fatalf("unexpected foreign kind %q", statement.Transactions[3].Kind)
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

const flexibleWithAppendixText = `CUENTA FLEXIBLE
Estado de Cuenta
NÚMERO DE CUENTACLABE INTERBANCARIA
6529009644021028065290096448
4Período del01022026 al 28022026
4Saldo Inicial del
Periodo
$ 9,858.21
DETALLE MOVIMIENTOS CUENTA FLEXIBLE No.  6529009644
DíaDescripciónReferencia
SerialRetiroCargoDepósitoAbonoSaldo
04A MI HSBC                        091225008045221
118423
$ 25,000.00 $ 34,858.21
04PAGO DE TARJETA: 5470749811846577 EN BPI13655983
41234
$ 30,000.00 $ 4,858.21
1524085327
Saldo Inicial $            9,858.21
InformaciónSPEI´s Recibidos durante el periodo del 01022026 al 28022026
1202202612:29:41BBVA BANCOMECLIENTE
00012028015289076108a mi hsbc$ 330,000.00MBAN0100
6583352`

const flexibleNumericHeaderText = `CUENTA FLEXIBLE
Estado de Cuenta
NÚMERO DE CUENTACLABE INTERBANCARIA
6529009644021028065290096448
4Período del01122025 al 31122025
4Saldo Inicial del
Periodo
$ 12,587.37
DETALLE MOVIMIENTOS CUENTA FLEXIBLE No.  6529009644
DíaDescripciónReferencia
SerialRetiroCargoDepósitoAbonoSaldo
24000012795364                     000012708045211
308379
$ 37,000.00 $ 49,587.37
Saldo Inicial $           12,587.37`

const ocrLikeFlexibleText = `CUENTA FLEXIBLE
Estado de Cuenta
HSBC
RESUMEN DE CUENTAS
Saldo Inicial del
$ 7,595.73
Periodo
Período de 01/11/2025 al 30/11/2025
DETALLE MOVIMIENTOS CUENTA FLEXIBLE No. 6529009644
Referencia/
Dia
Descripcion
Serial
Retiro/Cargo
Deposito/Abono
Saldo
18 AMI CUENTA HSBC
0811250
08045211
$ 25,000.00
$ 32,595.73
1704487
18 PAGO DE TARJETA: 547074981 1846577 EN BPI
13655983
$ 19,355.42
$ 13,240.31
41234
CoDi: Operacion procesada por CoDi®
Saldo Final $13,240.31`

const ocrLikeCardText = `HSBC 2Now Categoria: Oro
Numero de cuenta: 5470 7498 1184 6577
Periodo: 15-Dic-2025 al 12-Ene-2026
14-Dic-2025 15-Dic-2025 _ [SU PAGO GRACIAS $20,652.95
11-Dic-2025 15-Dic-2025 _|MERCADOPAGO *RAMEN664 Tijuana BCN + [$586,00
30-Dic-2025 30-Dic-2025 _NETFLIX MEXICO____CMX MEX + $249.00
03-Ene-2026 O7-Ene-2020 JANE 140618P37 AMAZON MEXICO __CIU +]$88.64
OPENAI *CHATGPT SUBSCR SAN FRANCISCO CA
08-Ene-2026 08-Ene-2026 MONEDA EXTRANJERA: 20.00 USD TC: 18.0705 DEL 08 DE ENERO + [8861.41
42-Ene-2026 12-Ene-2026 USO DE SALDO HSBC 2NOW +]$244 47
ATENCION DE QUEJAS`

const psm11LikeCardText = `HSBC 2Now Categoria: Oro
Numero de cuenta: 5470 7498 1184 6577
Periodo: 15-Dic-2025 al 12-Ene-2026
14-Dic-2025
15-Dic-2025
SU PAGO GRACIAS
-
$20,652.95
11-Dic-2025
15-Dic-2025
MERCADOPAGO *RAMEN664 Tijuana
BCN
+|$586.00
30-Dic-2025
30-Dic-2025
NETFLIX MEXICO
CMX
MEX
+|$249.00
OPENAI *CHATGPT SUBSCR SAN FRANCISCO CA
08-Ene-2026
08-Ene-2026
MONEDA EXTRANJERA:
20.00 USD TC: 18.0705 DEL 08 DE ENERO
+|$361.41
09-Ene-2026
12-Ene-2026
RSM _160408CSA DLO*SPOTIFY
MEX
+|$239.00
12-Ene-2026
412-Ene-2026
USO DE SALDO HSBC 2NOW
+|$244 47
ATENCION DE QUEJAS`

const psm11ArtifactCardText = `HSBC 2Now Categoria: Oro
Numero de cuenta: 5470 7498 1184 6577
Periodo: 13-Mar-2025 al 11-Abr-2025
12-Mar-2025
13-Mar-2025
MERCHANT ONE
Tl
+4$129.90
04-Abr-2025
07-Abr-2025
MERCADOPAGO *BORDERPL Tijuana
BCN
+
1,221.00
04-Abr-2025
07-Abr-2025
UPM 200220LK5 STRIPE *UBER TRIP
Clu
+1$284.90
OPENAI *CHATGPT SUBSCR SAN FRANCISCO CA
08-Abr-2025
08-Abr-2025
MONEDA EXTRANJERA:
20.00 USD TC: 20.79 DEL 08 DE ABRIL
+|$415.80
ATENCION DE QUEJAS`
