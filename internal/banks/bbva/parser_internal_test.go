package bbva

import (
	"testing"
	"time"

	edocuenta "github.com/DavidSerranoG/go-estado-cuenta-mx"
)

func mustParseRealTransactions(t *testing.T, text string, periodStart, periodEnd time.Time, wantCount int) []edocuenta.Transaction {
	t.Helper()

	transactions, warnings := parseRealTransactions(text, periodStart, periodEnd)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if len(transactions) != wantCount {
		t.Fatalf("len(transactions) = %d, want %d", len(transactions), wantCount)
	}

	return transactions
}

func sumDirections(transactions []edocuenta.Transaction) (debitSum, creditSum int64) {
	for _, tx := range transactions {
		switch tx.Direction {
		case edocuenta.TransactionDirectionDebit:
			debitSum += tx.AmountCents
		case edocuenta.TransactionDirectionCredit:
			creditSum += tx.AmountCents
		}
	}

	return debitSum, creditSum
}

func TestBBVAStatementHeaderSectionStopsBeforeMovementBody(t *testing.T) {
	t.Parallel()

	text := `BBVA
Información Financiera MONEDA NACIONAL
Detalle de Movimientos Realizados
CED CUENTA EN DOLARES`

	headerSection := bbvaStatementHeaderSection(text)
	if headerSection != "BBVA\nInformación Financiera MONEDA NACIONAL\n" {
		t.Fatalf("header section = %q", headerSection)
	}
}

func TestMatchBBVAStatementCurrencyUsesHeaderMarkers(t *testing.T) {
	t.Parallel()

	if got := matchBBVAStatementCurrency("Información Financiera MONEDA NACIONAL"); got != edocuenta.CurrencyMXN {
		t.Fatalf("national currency = %q, want %q", got, edocuenta.CurrencyMXN)
	}
	if got := matchBBVAStatementCurrency("Información Financiera MONEDA DOLARES"); got != edocuenta.CurrencyUSD {
		t.Fatalf("dollar currency = %q, want %q", got, edocuenta.CurrencyUSD)
	}
	if got := matchBBVAStatementCurrency("Libretón Dólares"); got != edocuenta.CurrencyUSD {
		t.Fatalf("dollar account label = %q, want %q", got, edocuenta.CurrencyUSD)
	}
}

func TestParseStatementCurrencyIgnoresDollarGlossaryOutsideHeader(t *testing.T) {
	t.Parallel()

	text := `BBVA
Información Financiera MONEDA NACIONAL
Detalle de Movimientos Realizados
TOTALES
CED CUENTA EN DOLARES
MN MONEDA NACIONAL`

	if got := parseStatementCurrency(text); got != edocuenta.CurrencyMXN {
		t.Fatalf("statement currency = %q, want %q", got, edocuenta.CurrencyMXN)
	}
}

func TestClassifyByDescriptionResolvesKnownStatementPatterns(t *testing.T) {
	t.Parallel()

	cases := map[string]edocuenta.TransactionDirection{
		"ABONO POR CORRECCION WWW ALIEXPRESS COM":  edocuenta.TransactionDirectionCredit,
		"ADYENMX*UBER EATS RFC: DEMO010101AAA":     edocuenta.TransactionDirectionDebit,
		"AMAZON MX MARKETPLACE RFC: DEMO010101AAA": edocuenta.TransactionDirectionDebit,
		"AMAZON MX RFC: DEMO010101AAA":             edocuenta.TransactionDirectionDebit,
		"SERV BANCA INTERNET":                      edocuenta.TransactionDirectionDebit,
		"IVA COM SERV BCA INTERNET":                edocuenta.TransactionDirectionDebit,
		"SU PAGO EN EFECTIVO":                      edocuenta.TransactionDirectionCredit,
		"COMISION POR MEMBRESIA":                   edocuenta.TransactionDirectionDebit,
		"BONIFICACION DE COMISION":                 edocuenta.TransactionDirectionCredit,
		"IVA COM MEMBRESIA":                        edocuenta.TransactionDirectionDebit,
		"BONIFICACION IVA COMISION":                edocuenta.TransactionDirectionCredit,
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
No. de Cuenta 0000000001

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
RFC: DEMO010101AAA 19:38 AUT: 248789 Referencia *****7339
21/MAR 21/MAR SPEI RECIBIDOBANAMEX 0 4,491.30 15,404.10 15,404.10
7910000PORTABILIDAD DE NOMINA Referencia 005324087 002
00005256782925183090
2019032140002NNNN0105797708092
CLIENTE,DEMO/UNO
22/MAR 22/MAR 2242095435TRANSFERENCIA PR 5.05 15,409.15 15,409.15
Referencia OP

Total de Movimientos
TOTAL IMPORTE CARGOS 5,087.20 TOTAL MOVIMIENTOS CARGOS 2
TOTAL IMPORTE ABONOS 20,496.35 TOTAL MOVIMIENTOS ABONOS 5`

	periodStart := time.Date(2019, 3, 9, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2019, 3, 22, 0, 0, 0, 0, time.UTC)

	transactions := mustParseRealTransactions(t, text, periodStart, periodEnd, 7)
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
No. de Cuenta 0000000002

Información Financiera MONEDA DOLARES
Saldo Anterior 25,444.87
Depósitos / Abonos (+) 2 4,787.83
Retiros / Cargos (-) 3 20,052.00
Saldo Final 10,180.70

Detalle de Movimientos Realizados
02/JUL   02/JUL PAGO CUENTA DE TERCERO                                                                      2,289.83       27,734.70          27,734.70
                 0000000001 BNET 0000000001 FACTURA DEMO1
16/JUL   16/JUL TRASPASO ENTRE CUENTAS                                                         20,000.00
                 5463155.1002.01 FOLIO: 0000000 355059.80MXP
16/JUL   16/JUL PAGO CUENTA DE TERCERO                                                                      2,498.00       10,232.70          10,180.70
                 0000000002 BNET 0000000001 FACTURA DEMO2
17/JUL   16/JUL SPO*QUARTYARD                                                                      22.00                   10,210.70          10,180.70
                 ******0434 USD 22.00TC001.0000AUT: 364009
18/JUL   16/JUL SQ *MEI SEMONES                                                                    30.00                   10,180.70          10,180.70
                 ******0434 USD 30.00TC001.0000AUT: 553261

Total de Movimientos
TOTAL IMPORTE CARGOS 20,052.00 TOTAL MOVIMIENTOS CARGOS 3
TOTAL IMPORTE ABONOS 4,787.83 TOTAL MOVIMIENTOS ABONOS 2`

	periodStart := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2025, 7, 31, 0, 0, 0, 0, time.UTC)

	transactions := mustParseRealTransactions(t, text, periodStart, periodEnd, 5)
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

func TestParseRealTransactionsResolvesLegacyMerchantDebitsAndBlizzardReversal(t *testing.T) {
	t.Parallel()

	text := `Estado de Cuenta
Libretón Básico
Periodo DEL 23/02/2021 AL 22/03/2021
No. de Cuenta 0000000001

Información Financiera MONEDA DOLARES
Saldo Anterior 9,759.79
Depósitos / Abonos (+) 4 50,449.88
Retiros / Cargos (-) 18 54,644.28
Saldo Final 5,565.39

Detalle de Movimientos Realizados
23/FEB 23/FEB GOOGLE *Domains 260.00 9,499.79 9,499.79
RFC: 13:12 AUT: 272732 Referencia ******6863
23/FEB 23/FEB TELNOR VENTA INT MU 1,038.00 8,461.79 8,461.79
RFC: DEMO010101AAA 13:17 AUT: 325592 Referencia ******6863
25/FEB 25/FEB SPEI RECIBIDOSCOTIABANK 23,200.00 31,661.79 31,661.79
0210225pago david febrero Referencia 0141751678 044
25/FEB 25/FEB PAGO TARJETA DE CREDITO 7,300.56 24,361.23 24,361.23
CUENTA: BMOV Referencia 3244313114
25/FEB 25/FEB SPEI RECIBIDOSCOTIABANK 23,200.00 47,561.23 47,561.23
0210225pago cliente demo Referencia 0141964163 044 00044000000000000000 2021022540044DEMO0000000000000 CLIENTE DEMO DOS
25/FEB 25/FEB SPEI ENVIADO HSBC 23,000.00 24,561.23 24,561.23
2502210A MI CUENTA DEMO Referencia 0087476647 021 00021028000000000000 MBAN01002102250087476647 CLIENTE DEMO UNO
25/FEB 25/FEB SPEI ENVIADO HSBC 15,000.00 9,561.23 9,561.23
2502210A MI CUENTA DEMO Referencia 0087947916 021 00021028000000000000 MBAN01002102260087947916 CLIENTE DEMO UNO
04/MAR 04/MAR Amazon web services 3.53 9,557.70 9,557.70
RFC: 06:17 AUT: 853238 Referencia ******6863
06/MAR 06/MAR PAGO CUENTA DE TERCERO 200.00 9,357.70 9,357.70
BNET 0000000003 PRESTAMO DEMO Referencia 4040585509
09/MAR 09/MAR 1PAYLU*RIOTGAMES 299.00 9,058.70 9,058.70
RFC: DEMO010101AAA 01:34 AUT: 420987 Referencia ******6863
09/MAR 09/MAR ADYENMX*UBER EATS 214.40 8,844.30 8,844.30
RFC: DEMO010101AAA 19:40 AUT: 467160 Referencia ******6863
16/MAR 12/MAR SERV GAS PREMIER 1,032.01
RFC: DEMO010101AAA 17:20 AUT: 704881 Referencia ******7339
16/MAR 13/MAR STEREN 595.00
RFC: DEMO010101AAA 15:04 AUT: 239134 Referencia ******7339
16/MAR 13/MAR FERRETERIA HIPODROMO 133.53
RFC: DEMO010101AAA 15:32 AUT: 605518 Referencia ******7339
16/MAR 13/MAR SUSHI BONSAI 270.00
RFC: DEMO010101AAA 15:57 AUT: 905111 Referencia ******7339
16/MAR 14/MAR OFFICE DEPOT DEMO 491.37
RFC: DEMO010101AAA 18:44 AUT: 417909 Referencia ******7339
16/MAR 16/MAR BLIZZARD ENTERTAINM 1,049.88 5,272.51 2,505.51
RFC: 02:17 AUT: 352015 Referencia ******6863
17/MAR 16/MAR BP*REFACCION 2,698.00
RFC: DEMO010101AAA 15:21 AUT: 401194 Referencia ******7339
17/MAR 16/MAR STEREN 69.00
RFC: DEMO010101AAA 16:10 AUT: 887722 Referencia ******7339
17/MAR 17/MAR BLIZZARD ENTERTAINM 1,049.88 3,555.39 3,555.39
RFC: 00:00 AUT: Referencia ******6863
22/MAR 20/MAR SUSHI BONSAI 990.00
RFC: DEMO010101AAA 15:38 AUT: 453570 Referencia ******7339
22/MAR 22/MAR SPEI RECIBIDOHSBC 3,000.00 5,565.39 5,565.39
0000001A MI CUENTA DEMO Referencia 0179306988 021

Total de Movimientos
TOTAL IMPORTE CARGOS 54,644.28 TOTAL MOVIMIENTOS CARGOS 18
TOTAL IMPORTE ABONOS 50,449.88 TOTAL MOVIMIENTOS ABONOS 4`

	periodStart := time.Date(2021, 2, 23, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2021, 3, 22, 0, 0, 0, 0, time.UTC)

	transactions := mustParseRealTransactions(t, text, periodStart, periodEnd, 22)

	var debitSum, creditSum int64
	var firstBlizzard, secondBlizzard *edocuenta.Transaction
	for i := range transactions {
		switch transactions[i].Direction {
		case edocuenta.TransactionDirectionDebit:
			debitSum += transactions[i].AmountCents
		case edocuenta.TransactionDirectionCredit:
			creditSum += transactions[i].AmountCents
		}
		if transactions[i].Description == "BLIZZARD ENTERTAINM" {
			if firstBlizzard == nil {
				firstBlizzard = &transactions[i]
			} else {
				secondBlizzard = &transactions[i]
			}
		}
	}

	if debitSum != 5464428 {
		t.Fatalf("debitSum = %d, want 5464428", debitSum)
	}
	if creditSum != 5044988 {
		t.Fatalf("creditSum = %d, want 5044988", creditSum)
	}
	if firstBlizzard == nil || secondBlizzard == nil {
		t.Fatalf("expected both BLIZZARD transactions, got first=%v second=%v", firstBlizzard, secondBlizzard)
	}
	if firstBlizzard.Direction != edocuenta.TransactionDirectionDebit {
		t.Fatalf("first BLIZZARD direction = %q, want debit", firstBlizzard.Direction)
	}
	if secondBlizzard.Direction != edocuenta.TransactionDirectionCredit {
		t.Fatalf("second BLIZZARD direction = %q, want credit", secondBlizzard.Direction)
	}
	if secondBlizzard.AmountCents != 104988 {
		t.Fatalf("second BLIZZARD amount = %d, want 104988", secondBlizzard.AmountCents)
	}
}

func TestParseRealTransactionsInfersThreeCreditsAcrossTrailingBalance(t *testing.T) {
	t.Parallel()

	text := `Estado de Cuenta
Libretón Básico Cuenta Digital
Periodo DEL 23/09/2024 AL 22/10/2024
No. de Cuenta 0000000001

Información Financiera MONEDA NACIONAL
Saldo Anterior 291,774.54
Depósitos / Abonos (+) 4 20,850.00
Retiros / Cargos (-) 2 45,951.93
Saldo Final 266,672.61

Detalle de Movimientos Realizados
23/SEP       23/SEP        SPEI RECIBIDOHSBC                                                                              20,000.00      311,774.54              311,774.54
                           0000001A mi BBVA                                               Referencia 0132887483 021
27/SEP       27/SEP        PAGO TARJETA DE CREDITO                                                         25,000.00
                           CUENTA: BMOV                                                   Referencia 6415654409
27/SEP       27/SEP        PAGO TARJETA DE CREDITO                                                         20,951.93                  265,822.61       265,822.61
                           CUENTA: BMOV                                                  Referencia 6415924890
28/SEP       30/SEP        PAGO CUENTA DE TERCERO                                                                           233.00
                           BNET 1566472338 desayuno                                      Referencia 6520342637
28/SEP       30/SEP        PAGO CUENTA DE TERCERO                                                                           267.00
                           BNET 1530642646 Transf a undefined                            Referencia 6520373014
28/SEP       30/SEP        PAGO CUENTA DE TERCERO                                                                           350.00    266,672.61       265,822.61
                           BNET 1553889228 Gracias bb                                    Referencia 6520381424

Total de Movimientos
TOTAL IMPORTE CARGOS 45,951.93 TOTAL MOVIMIENTOS CARGOS 2
TOTAL IMPORTE ABONOS 20,850.00 TOTAL MOVIMIENTOS ABONOS 4`

	periodStart := time.Date(2024, 9, 23, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC)

	transactions := mustParseRealTransactions(t, text, periodStart, periodEnd, 6)

	for i := 3; i < 6; i++ {
		if transactions[i].Description != "PAGO CUENTA DE TERCERO" {
			t.Fatalf("transactions[%d].Description = %q, want PAGO CUENTA DE TERCERO", i, transactions[i].Description)
		}
		if transactions[i].Direction != edocuenta.TransactionDirectionCredit {
			t.Fatalf("transactions[%d].Direction = %q, want credit", i, transactions[i].Direction)
		}
	}
	if transactions[5].AmountCents != 35000 {
		t.Fatalf("last third-party amount = %d, want 35000", transactions[5].AmountCents)
	}
	if transactions[5].BalanceCents == nil || *transactions[5].BalanceCents != 26667261 {
		t.Fatalf("last third-party balance = %v, want 26667261", transactions[5].BalanceCents)
	}
}

func TestParseRealTransactionsInfersMixedSpanBeforeKnownMerchantBalance(t *testing.T) {
	t.Parallel()

	text := `Estado de Cuenta
Libretón Básico Cuenta Digital
Periodo DEL 23/02/2023 AL 22/03/2023
No. de Cuenta 0000000001

Información Financiera MONEDA NACIONAL
Saldo Anterior 102,752.44
Depósitos / Abonos (+) 2 58,286.70
Retiros / Cargos (-) 3 50,179.00
Saldo Final 110,860.14

Detalle de Movimientos Realizados
23/FEB       23/FEB        SPEI ENVIADO HSBC                                                               11,000.00                     91,752.44            83,753.44
                           2302230A mi HSBC                                               Referencia 0095720271 021
24/FEB       24/FEB        PAGO CUENTA DE TERCERO                                                                         29,143.35
                           BNET 0000000004 FACTURA DEMO3                                  Referencia 0070246047
24/FEB       24/FEB        SPEI ENVIADO HSBC                                                               32,000.00
                           2402230A mi HSBC                                              Referencia 0099516853 021
24/FEB       23/FEB        COMERCIO DEMO MAYORISTA                                                             7,999.00               80,896.79        80,896.79
                           RFC: CME 910715UB9 19:49 AUT: 218016                          Referencia ******7339
10/MAR       10/MAR        PAGO CUENTA DE TERCERO                                                                         29,143.35   110,040.14       110,040.14
                           BNET 0000000004 FACTURA DEMO4                                 Referencia 0082861026
17/MAR       17/MAR        RETIRO SIN TARJETA                                                                  3,000.00
                                                                                          Referencia ******7339
17/MAR       17/MAR        SAT                                                                                 1,180.00               105,860.14       105,860.14
                           REF:04231JWO020038037228 CIE:0844985                          Referencia GUIA:1112705

Total de Movimientos
TOTAL IMPORTE CARGOS 50,179.00 TOTAL MOVIMIENTOS CARGOS 3
TOTAL IMPORTE ABONOS 58,286.70 TOTAL MOVIMIENTOS ABONOS 2`

	periodStart := time.Date(2023, 2, 23, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2023, 3, 22, 0, 0, 0, 0, time.UTC)

	transactions := mustParseRealTransactions(t, text, periodStart, periodEnd, 7)
	if transactions[1].Description != "PAGO CUENTA DE TERCERO" || transactions[1].Direction != edocuenta.TransactionDirectionCredit {
		t.Fatalf("transactions[1] = (%q,%q), want (PAGO CUENTA DE TERCERO,credit)", transactions[1].Description, transactions[1].Direction)
	}
	if transactions[2].Description != "SPEI ENVIADO HSBC" || transactions[2].Direction != edocuenta.TransactionDirectionDebit {
		t.Fatalf("transactions[2] = (%q,%q), want (SPEI ENVIADO HSBC,debit)", transactions[2].Description, transactions[2].Direction)
	}
	if transactions[3].Description != "COMERCIO DEMO MAYORISTA" || transactions[3].Direction != edocuenta.TransactionDirectionDebit {
		t.Fatalf("transactions[3] = (%q,%q), want (COMERCIO DEMO MAYORISTA,debit)", transactions[3].Description, transactions[3].Direction)
	}
	if transactions[3].AmountCents != 799900 {
		t.Fatalf("transactions[3].AmountCents = %d, want 799900", transactions[3].AmountCents)
	}
	if transactions[4].Description != "PAGO CUENTA DE TERCERO" || transactions[4].Direction != edocuenta.TransactionDirectionCredit {
		t.Fatalf("transactions[4] = (%q,%q), want (PAGO CUENTA DE TERCERO,credit)", transactions[4].Description, transactions[4].Direction)
	}
}

func TestParseRealTransactionsResolvesLegacyBankingFeesAndCashPayments(t *testing.T) {
	t.Parallel()

	text := `Estado de Cuenta
Libretón Básico
Periodo DEL 23/08/2020 AL 22/09/2020
No. de Cuenta 0000000001

Información Financiera MONEDA NACIONAL
Saldo Anterior 1,000.00
Depósitos / Abonos (+) 2 4,960.00
Retiros / Cargos (-) 2 5.80
Saldo Final 5,954.20

Detalle de Movimientos Realizados
04/SEP       04/SEP        SERV BANCA INTERNET                                                                    5.00
                                                                                         Referencia OPS SERV BCA IN
04/SEP       04/SEP        IVA COM SERV BCA INTERNET                                                              0.80
                                                                                         Referencia IVA COM SERV BC
11/SEP       11/SEP        SU PAGO EN EFECTIVO                                                                             3,600.00   4,594.20            4,594.20
                           EN COMERCIO
18/SEP       18/SEP        SU PAGO EN EFECTIVO                                                                             1,360.00   5,954.20            5,954.20
                           EN COMERCIO

Total de Movimientos
TOTAL IMPORTE CARGOS 5.80 TOTAL MOVIMIENTOS CARGOS 2
TOTAL IMPORTE ABONOS 4,960.00 TOTAL MOVIMIENTOS ABONOS 2`

	periodStart := time.Date(2020, 8, 23, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2020, 9, 22, 0, 0, 0, 0, time.UTC)

	transactions := mustParseRealTransactions(t, text, periodStart, periodEnd, 4)
	debitSum, creditSum := sumDirections(transactions)
	if debitSum != 580 {
		t.Fatalf("debitSum = %d, want 580", debitSum)
	}
	if creditSum != 496000 {
		t.Fatalf("creditSum = %d, want 496000", creditSum)
	}
}

func TestParseRealTransactionsResolvesMembershipChargesAndReversals(t *testing.T) {
	t.Parallel()

	text := `Estado de Cuenta
Libretón Básico
Periodo DEL 23/01/2022 AL 22/02/2022
No. de Cuenta 0000000001

Información Financiera MONEDA NACIONAL
Saldo Anterior 65,013.18
Depósitos / Abonos (+) 2 63.80
Retiros / Cargos (-) 2 63.80
Saldo Final 65,013.18

Detalle de Movimientos Realizados
24/ENE       23/ENE        COMISION POR MEMBRESIA                                                                55.00
                           POR MANTENER SALDO INFERIOR AL MINIMO                          Referencia 23DIC21/22ENE22
24/ENE       23/ENE        BONIFICACION DE COMISION                                                                        55.00
                           COMISION POR MEMBRESIA                                         Referencia 23DIC21/22ENE22
24/ENE       23/ENE        IVA COM MEMBRESIA                                                                      8.80
                           16%
24/ENE       23/ENE        BONIFICACION IVA COMISION                                                                           8.80   65,013.18        65,013.18

Total de Movimientos
TOTAL IMPORTE CARGOS 63.80 TOTAL MOVIMIENTOS CARGOS 2
TOTAL IMPORTE ABONOS 63.80 TOTAL MOVIMIENTOS ABONOS 2`

	periodStart := time.Date(2022, 1, 23, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2022, 2, 22, 0, 0, 0, 0, time.UTC)

	transactions := mustParseRealTransactions(t, text, periodStart, periodEnd, 4)

	wantKinds := []edocuenta.TransactionDirection{
		edocuenta.TransactionDirectionDebit,
		edocuenta.TransactionDirectionCredit,
		edocuenta.TransactionDirectionDebit,
		edocuenta.TransactionDirectionCredit,
	}
	for i, want := range wantKinds {
		if transactions[i].Direction != want {
			t.Fatalf("transactions[%d].Direction = %q, want %q", i, transactions[i].Direction, want)
		}
	}
}

func TestParseRealTransactionsBalanceOverridesDebitHintWhenBalanceIncreases(t *testing.T) {
	t.Parallel()

	text := `Estado de Cuenta
Libretón Básico
Periodo DEL 01/12/2022 AL 31/12/2022
No. de Cuenta 0000000001

Información Financiera MONEDA NACIONAL
Saldo Anterior 44,458.60
Depósitos / Abonos (+) 1 45,286.70
Retiros / Cargos (-) 0 0.00
Saldo Final 89,745.30

Detalle de Movimientos Realizados
16/DIC       16/DIC        SPEI ENVIADO HSBC                                                                    45,286.70              89,745.30           89,745.30
                           Referencia 0001

Total de Movimientos
TOTAL IMPORTE CARGOS 0.00 TOTAL MOVIMIENTOS CARGOS 0
TOTAL IMPORTE ABONOS 45,286.70 TOTAL MOVIMIENTOS ABONOS 1`

	periodStart := time.Date(2022, 12, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2022, 12, 31, 0, 0, 0, 0, time.UTC)

	transactions := mustParseRealTransactions(t, text, periodStart, periodEnd, 1)
	if transactions[0].Direction != edocuenta.TransactionDirectionCredit {
		t.Fatalf("direction = %q, want credit", transactions[0].Direction)
	}
	if transactions[0].AmountCents != 4528670 {
		t.Fatalf("amount = %d, want 4528670", transactions[0].AmountCents)
	}
}

func TestParseRealTransactionsBalanceOverridesDebitHintForThirdPartyPayment(t *testing.T) {
	t.Parallel()

	text := `Estado de Cuenta
Libretón Básico
Periodo DEL 01/02/2022 AL 28/02/2022
No. de Cuenta 0000000001

Información Financiera MONEDA NACIONAL
Saldo Anterior 37,972.95
Depósitos / Abonos (+) 1 675.00
Retiros / Cargos (-) 0 0.00
Saldo Final 38,647.95

Detalle de Movimientos Realizados
07/FEB       07/FEB        PAGO CUENTA DE TERCERO                                                                  675.00               38,647.95           38,647.95
                           Referencia 0001

Total de Movimientos
TOTAL IMPORTE CARGOS 0.00 TOTAL MOVIMIENTOS CARGOS 0
TOTAL IMPORTE ABONOS 675.00 TOTAL MOVIMIENTOS ABONOS 1`

	periodStart := time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2022, 2, 28, 0, 0, 0, 0, time.UTC)

	transactions := mustParseRealTransactions(t, text, periodStart, periodEnd, 1)
	if transactions[0].Direction != edocuenta.TransactionDirectionCredit {
		t.Fatalf("direction = %q, want credit", transactions[0].Direction)
	}
	if transactions[0].AmountCents != 67500 {
		t.Fatalf("amount = %d, want 67500", transactions[0].AmountCents)
	}
}

func TestParseRealTransactionsInfersCreditThenDebitPairFromNextBalance(t *testing.T) {
	t.Parallel()

	text := `Estado de Cuenta
Libretón Básico
Periodo DEL 01/12/2022 AL 31/12/2022
No. de Cuenta 0000000001

Información Financiera MONEDA NACIONAL
Saldo Anterior 73,601.95
Depósitos / Abonos (+) 1 29,143.35
Retiros / Cargos (-) 1 13,000.00
Saldo Final 89,745.30

Detalle de Movimientos Realizados
16/DIC       16/DIC        PAGO CUENTA DE TERCERO                                                                         29,143.35
                           BNET 0000000004 FACTURA DEMO5                                 Referencia 0076933010
16/DIC       16/DIC        SPEI ENVIADO HSBC                                                               13,000.00                  89,745.30        89,745.30
                           1612220A MI CUENTA DEMO                                        Referencia 0094972896 021

Total de Movimientos
TOTAL IMPORTE CARGOS 13,000.00 TOTAL MOVIMIENTOS CARGOS 1
TOTAL IMPORTE ABONOS 29,143.35 TOTAL MOVIMIENTOS ABONOS 1`

	periodStart := time.Date(2022, 12, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2022, 12, 31, 0, 0, 0, 0, time.UTC)

	transactions := mustParseRealTransactions(t, text, periodStart, periodEnd, 2)
	if transactions[0].Direction != edocuenta.TransactionDirectionCredit {
		t.Fatalf("first direction = %q, want credit", transactions[0].Direction)
	}
	if transactions[0].AmountCents != 2914335 {
		t.Fatalf("first amount = %d, want 2914335", transactions[0].AmountCents)
	}
	if transactions[1].Direction != edocuenta.TransactionDirectionDebit {
		t.Fatalf("second direction = %q, want debit", transactions[1].Direction)
	}
	if transactions[1].AmountCents != 1300000 {
		t.Fatalf("second amount = %d, want 1300000", transactions[1].AmountCents)
	}
}

func TestParseRealTransactionsInfersPairAfterEarlierHintOnlyRows(t *testing.T) {
	t.Parallel()

	text := `Estado de Cuenta
Libretón Básico
Periodo DEL 23/01/2022 AL 22/02/2022
No. de Cuenta 0000000001

Información Financiera MONEDA NACIONAL
Saldo Anterior 65,013.18
Depósitos / Abonos (+) 4 49,728.42
Retiros / Cargos (-) 4 73,137.51
Saldo Final 41,604.09

Detalle de Movimientos Realizados
24/ENE       23/ENE        COMISION POR MEMBRESIA                                                                55.00
                           POR MANTENER SALDO INFERIOR AL MINIMO                          Referencia 23DIC21/22ENE22
24/ENE       23/ENE        BONIFICACION DE COMISION                                                                        55.00
                           COMISION POR MEMBRESIA                                         Referencia 23DIC21/22ENE22
24/ENE       23/ENE        IVA COM MEMBRESIA                                                                      8.80
                           16%
24/ENE       23/ENE        BONIFICACION IVA COMISION                                                                           8.80   65,013.18        65,013.18
28/ENE       28/ENE        PAGO CUENTA DE TERCERO                                                                         23,418.77
                           BNET 0000000004 FACTURA DEMO6                                 Referencia 0035741018
28/ENE       28/ENE        SPEI ENVIADO HSBC                                                               50,000.00                  38,431.95        38,431.95
                           2801220A MI CUENTA DEMO                                        Referencia 0077363373 021

Total de Movimientos
TOTAL IMPORTE CARGOS 50,063.80 TOTAL MOVIMIENTOS CARGOS 4
TOTAL IMPORTE ABONOS 23,482.57 TOTAL MOVIMIENTOS ABONOS 4`

	periodStart := time.Date(2022, 1, 23, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2022, 2, 22, 0, 0, 0, 0, time.UTC)

	transactions := mustParseRealTransactions(t, text, periodStart, periodEnd, 6)
	if transactions[4].Direction != edocuenta.TransactionDirectionCredit {
		t.Fatalf("fifth direction = %q, want credit", transactions[4].Direction)
	}
	if transactions[4].AmountCents != 2341877 {
		t.Fatalf("fifth amount = %d, want 2341877", transactions[4].AmountCents)
	}
	if transactions[5].Direction != edocuenta.TransactionDirectionDebit {
		t.Fatalf("sixth direction = %q, want debit", transactions[5].Direction)
	}
	if transactions[5].AmountCents != 5000000 {
		t.Fatalf("sixth amount = %d, want 5000000", transactions[5].AmountCents)
	}
}

func TestParseRealTransactionsInfersTwoCreditsWhenNextHintConflicts(t *testing.T) {
	t.Parallel()

	text := `Estado de Cuenta
Libretón Básico
Periodo DEL 01/02/2022 AL 28/02/2022
No. de Cuenta 0000000001

Información Financiera MONEDA NACIONAL
Saldo Anterior 38,197.95
Depósitos / Abonos (+) 2 450.00
Retiros / Cargos (-) 0 0.00
Saldo Final 38,647.95

Detalle de Movimientos Realizados
07/FEB       08/FEB        PAGO CUENTA DE TERCERO                                                                           225.00
                           BNET 1580631760 AGUA                                          Referencia 3231495370
07/FEB       08/FEB        PAGO CUENTA DE TERCERO                                                                           225.00    38,647.95        38,197.95
                           BNET 1558072949 AGUA                                          Referencia 3231534118

Total de Movimientos
TOTAL IMPORTE CARGOS 0.00 TOTAL MOVIMIENTOS CARGOS 0
TOTAL IMPORTE ABONOS 450.00 TOTAL MOVIMIENTOS ABONOS 2`

	periodStart := time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2022, 2, 28, 0, 0, 0, 0, time.UTC)

	transactions := mustParseRealTransactions(t, text, periodStart, periodEnd, 2)
	for i, tx := range transactions {
		if tx.Direction != edocuenta.TransactionDirectionCredit {
			t.Fatalf("transactions[%d].Direction = %q, want credit", i, tx.Direction)
		}
		if tx.AmountCents != 22500 {
			t.Fatalf("transactions[%d].AmountCents = %d, want 22500", i, tx.AmountCents)
		}
	}
}
