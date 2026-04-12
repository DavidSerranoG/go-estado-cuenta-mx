package edocuenta_test

import (
	"context"
	"errors"
	"testing"

	edocuenta "github.com/DavidSerranoG/go-estado-cuenta-mx"
	"github.com/DavidSerranoG/go-estado-cuenta-mx/bbva"
	"github.com/DavidSerranoG/go-estado-cuenta-mx/hsbc"
)

func TestProcessorRejectsMissingParsers(t *testing.T) {
	t.Parallel()

	processor := edocuenta.New()

	_, err := processor.ParseText("HSBC MEXICO")
	if !errors.Is(err, edocuenta.ErrNoParsersConfigured) {
		t.Fatalf("expected ErrNoParsersConfigured, got %v", err)
	}
}

func TestProcessorDetectsHSBCText(t *testing.T) {
	t.Parallel()

	processor := edocuenta.New(
		edocuenta.WithParser(hsbc.New()),
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

	result, err := processor.ParseTextResult(sampleHSBCText)
	if err != nil {
		t.Fatalf("parse text result: %v", err)
	}
	if result.Diagnostics.Layout != "card" {
		t.Fatalf("expected card layout, got %q", result.Diagnostics.Layout)
	}
	if result.Diagnostics.Confidence != edocuenta.ParseConfidenceHigh {
		t.Fatalf("expected high confidence, got %q", result.Diagnostics.Confidence)
	}
}

func TestProcessorPrefersStructuralDetectionOverBankMentionsInBody(t *testing.T) {
	t.Parallel()

	processor := edocuenta.New(
		edocuenta.WithParser(bbva.New()),
		edocuenta.WithParser(hsbc.New()),
	)

	statement, err := processor.ParseText(sampleFlexibleHSBCWithBBVAMention)
	if err != nil {
		t.Fatalf("parse text: %v", err)
	}

	if statement.Bank != "hsbc" {
		t.Fatalf("expected hsbc parser to win, got %q", statement.Bank)
	}

	result, err := processor.ParseTextResult(sampleFlexibleHSBCWithBBVAMention)
	if err != nil {
		t.Fatalf("parse text result: %v", err)
	}
	if result.Diagnostics.Layout != "flexible" {
		t.Fatalf("expected flexible layout, got %q", result.Diagnostics.Layout)
	}
}

func TestProcessorRetriesWithRescueExtractorWhenBBVACardTextIsIncomplete(t *testing.T) {
	t.Parallel()

	rescueCalls := 0
	processor := edocuenta.New(
		edocuenta.WithExtractor(staticExtractor{text: degradedBBVACardText}),
		edocuenta.WithRescueExtractor(countingExtractor{text: rescuedBBVACardText, called: &rescueCalls}),
		edocuenta.WithParser(bbva.New()),
	)

	result, err := processor.ParsePDFResult(context.Background(), []byte("pdf"))
	if err != nil {
		t.Fatalf("parse pdf with rescue extractor: %v", err)
	}
	statement := result.Statement
	if rescueCalls != 1 {
		t.Fatalf("expected rescue extractor to run once, got %d", rescueCalls)
	}
	if statement.Bank != "bbva" {
		t.Fatalf("unexpected bank %q", statement.Bank)
	}
	if statement.AccountNumber != "XXXXXX9919" {
		t.Fatalf("unexpected account %q", statement.AccountNumber)
	}
	if len(statement.Transactions) != 3 {
		t.Fatalf("expected 3 transactions, got %d", len(statement.Transactions))
	}
	if result.ExtractedText != rescuedBBVACardText {
		t.Fatalf("expected rescued extracted text, got %q", result.ExtractedText)
	}
	if result.Extraction.SelectedExtractor != "rescue" || !result.Extraction.UsedRescue {
		t.Fatalf("unexpected extraction diagnostics %+v", result.Extraction)
	}
	if result.Diagnostics.Layout != "card" {
		t.Fatalf("expected card layout, got %q", result.Diagnostics.Layout)
	}
	if result.Diagnostics.Confidence != edocuenta.ParseConfidenceHigh {
		t.Fatalf("expected high confidence, got %q", result.Diagnostics.Confidence)
	}
}

func TestProcessorPreservesOriginalErrorWhenRescueExtractorFails(t *testing.T) {
	t.Parallel()

	originalErr := mustParseBBVACardError(t, degradedBBVACardText)

	processor := edocuenta.New(
		edocuenta.WithExtractor(staticExtractor{text: degradedBBVACardText}),
		edocuenta.WithRescueExtractor(errExtractor{err: errors.New("tesseract unavailable")}),
		edocuenta.WithParser(bbva.New()),
	)

	_, err := processor.ParsePDF(context.Background(), []byte("pdf"))
	if err == nil {
		t.Fatal("expected parse error")
	}
	if err.Error() != originalErr.Error() {
		t.Fatalf("expected original error %q, got %q", originalErr.Error(), err.Error())
	}
}

const sampleHSBCText = `HSBC MEXICO
NÚMERO DE CUENTA: 5470 7498 1184 6577
TU PAGO REQUERIDO ESTE PERIODO
15-Sep-2025 al 12-Oct-2025
16-Sep-202517-Sep-2025SU PAGO GRACIAS-         $25,000.00
12-Sep-202515-Sep-2025RUGR590104PR9 SERV GAS PREMIER       TIJ+$868.07`

const sampleFlexibleHSBCWithBBVAMention = `CUENTA FLEXIBLE
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
InformaciónSPEI´s Recibidos durante el periodo del 01022026 al 28022026
1202202612:29:41BBVA BANCOMERCLIENTE
00012028015289076108a mi hsbc$ 330,000.00MBAN0100`

const degradedBBVACardText = `BBVA
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

const rescuedBBVACardText = `BBVA
TARJETA AZUL BBVA (CLASICA)
Número de tarjeta: 4772913064069919
Periodo: 25-feb-2026 al 24-mar-2026
DESGLOSE DE MOVIMIENTOS
CARGOS,COMPRAS Y ABONOS REGULARES(NO A MESES) Tarjeta titular: XXXXXXXXXXXX9919
04-mar-2026 04-mar-2026 BMOVIL.PAGO TDC - $15,297.64
Número de cuenta: XXXXXX9919
04-mar-2026 06-mar-2026 AMAZON WEB SERVICES ; Tarjeta Digital ***2932 + $218.68
15-mar-2026 17-mar-2026 MVBILLING.COM ; Tarjeta Digital ***2932 + $194.56
MXP $194.56 TIPO DE CAMBIO $1.00
TOTAL CARGOS $413.24
TOTAL ABONOS -$15,297.64
ATENCION DE QUEJAS`

type countingExtractor struct {
	text   string
	called *int
}

func (e countingExtractor) Name() string {
	return "rescue"
}

func (e countingExtractor) ExtractText(context.Context, []byte) (string, error) {
	if e.called != nil {
		*e.called++
	}
	return e.text, nil
}

type errExtractor struct {
	err error
}

func (e errExtractor) Name() string {
	return "broken-rescue"
}

func (e errExtractor) ExtractText(context.Context, []byte) (string, error) {
	return "", e.err
}

func mustParseBBVACardError(t *testing.T, text string) error {
	t.Helper()

	_, err := bbva.New().Parse(text)
	if err == nil {
		t.Fatal("expected bbva parse error")
	}

	return err
}
