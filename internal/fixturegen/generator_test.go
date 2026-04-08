package fixturegen

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	edocuenta "github.com/DavidSerranoG/go-estado-cuenta-mx"
	"github.com/DavidSerranoG/go-estado-cuenta-mx/supported"
	"github.com/go-pdf/fpdf"
)

func TestGenerateFileProducesParseableDummyPDF(t *testing.T) {
	t.Parallel()

	lines := []string{
		"BBVA",
		"TARJETA AZUL BBVA (CLASICA)",
		"Numero de tarjeta: 4772913064069919",
		"Periodo: 25-feb-2026 al 24-mar-2026",
		"DESGLOSE DE MOVIMIENTOS",
		"CARGOS,COMPRAS Y ABONOS REGULARES(NO A MESES) Tarjeta titular: XXXXXXXXXXXX9919",
		"04-mar-2026 04-mar-2026 BMOVIL.PAGO TDC - $15,297.64",
		"Numero de cuenta: XXXXXX9919",
		"04-mar-2026 06-mar-2026 AMAZON WEB SERVICES ; Tarjeta Digital ***2932 + $218.68",
		"15-mar-2026 17-mar-2026 MVBILLING.COM ; Tarjeta Digital ***2932 + $194.56",
		"MXP $194.56 TIPO DE CAMBIO $1.00",
		"TOTAL CARGOS $413.24",
		"TOTAL ABONOS -$15,297.64",
		"ATENCION DE QUEJAS",
	}

	pdfBytes := mustPDF(t, lines)
	inputPath := filepath.Join(t.TempDir(), "bbva-card.pdf")
	if err := os.WriteFile(inputPath, pdfBytes, 0o600); err != nil {
		t.Fatalf("write input pdf: %v", err)
	}

	rasterPath := mustBlankPNG(t, 1224, 1584)
	generator := &Generator{
		processor:     supported.New(),
		bboxExtractor: stubBBoxExtractor{doc: textDocFromLines(lines)},
		rasterizer:    stubRasterizer{doc: RasterDocument{Tool: "stub-raster", Pages: []RasterPage{{Path: rasterPath, Width: 612, Height: 792}}}},
		overridesRoot: filepath.Join("testdata", "fixturegen", "overrides"),
	}

	metadata, err := generator.GenerateFile(context.Background(), inputPath, Config{
		Output:   t.TempDir(),
		Bank:     "bbva",
		Layout:   "card",
		Branding: BrandingMixed,
		Mode:     OutputPublic,
	})
	if err != nil {
		t.Fatalf("generate file: %v", err)
	}
	if !metadata.Validation.ParseOK {
		t.Fatalf("expected parseable dummy, got %+v", metadata.Validation)
	}
	if metadata.Fidelity != FidelityHigh {
		t.Fatalf("expected high fidelity, got %q", metadata.Fidelity)
	}
	if metadata.OutputPath == "" || metadata.SidecarPath == "" {
		t.Fatalf("expected output paths in metadata, got %+v", metadata)
	}

	dummyBytes, err := os.ReadFile(metadata.OutputPath)
	if err != nil {
		t.Fatalf("read dummy pdf: %v", err)
	}
	result, err := supported.New().ParsePDFResult(context.Background(), dummyBytes)
	if err != nil {
		t.Fatalf("parse dummy pdf: %v", err)
	}
	if len(result.Statement.Transactions) != 3 {
		t.Fatalf("expected 3 transactions, got %d", len(result.Statement.Transactions))
	}
	if strings.Contains(result.ExtractedText, "4772913064069919") {
		t.Fatalf("dummy extracted text leaked original card number")
	}
	if strings.Contains(result.ExtractedText, "AMAZON WEB SERVICES") {
		t.Fatalf("dummy extracted text leaked original merchant name")
	}
}

func TestGenerateFileCheckModeDoesNotWriteFiles(t *testing.T) {
	t.Parallel()

	lines := []string{
		"HSBC MEXICO",
		"NUMERO DE CUENTA: 5470 7498 1184 6577",
		"TU PAGO REQUERIDO ESTE PERIODO",
		"15-Sep-2025 al 12-Oct-2025",
		"16-Sep-2025 17-Sep-2025 SU PAGO GRACIAS - $25,000.00",
		"12-Sep-2025 15-Sep-2025 SERV GAS PREMIER TIJ + $868.07",
	}

	pdfBytes := mustPDF(t, lines)
	inputPath := filepath.Join(t.TempDir(), "hsbc-card.pdf")
	if err := os.WriteFile(inputPath, pdfBytes, 0o600); err != nil {
		t.Fatalf("write input pdf: %v", err)
	}

	rasterPath := mustBlankPNG(t, 1224, 1584)
	generator := &Generator{
		processor:     supported.New(),
		bboxExtractor: stubBBoxExtractor{doc: textDocFromLines(lines)},
		rasterizer:    stubRasterizer{doc: RasterDocument{Tool: "stub-raster", Pages: []RasterPage{{Path: rasterPath, Width: 612, Height: 792}}}},
	}

	outputRoot := t.TempDir()
	metadata, err := generator.GenerateFile(context.Background(), inputPath, Config{
		Output:   outputRoot,
		Bank:     "hsbc",
		Layout:   "card",
		Branding: BrandingMixed,
		Mode:     OutputPublic,
		Check:    true,
	})
	if err != nil {
		t.Fatalf("generate file in check mode: %v", err)
	}
	if metadata.OutputPath != "" || metadata.SidecarPath != "" {
		t.Fatalf("expected no output paths in check mode, got %+v", metadata)
	}
	entries, err := os.ReadDir(outputRoot)
	if err != nil {
		t.Fatalf("read output dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no files written in check mode, got %d", len(entries))
	}
}

func TestGenerateFileRejectsLowFidelityInPublicMode(t *testing.T) {
	t.Parallel()

	lines := []string{
		"BBVA",
		"Cuenta: 1234567890",
		"Periodo: 01/03/2026 - 31/03/2026",
		"01/03/2026 DEPOSITO ABONO 1,000.00 10,000.00",
	}

	pdfBytes := mustPDF(t, lines)
	inputPath := filepath.Join(t.TempDir(), "bbva-account.pdf")
	if err := os.WriteFile(inputPath, pdfBytes, 0o600); err != nil {
		t.Fatalf("write input pdf: %v", err)
	}

	generator := &Generator{
		processor:     supported.New(),
		bboxExtractor: stubBBoxExtractor{doc: textDocFromLines(lines)},
		rasterizer:    stubRasterizer{err: errRasterUnavailable},
	}

	_, err := generator.GenerateFile(context.Background(), inputPath, Config{
		Output:   t.TempDir(),
		Bank:     "bbva",
		Layout:   "account",
		Branding: BrandingMixed,
		Mode:     OutputPublic,
	})
	if err == nil || !strings.Contains(err.Error(), "public mode requires high-fidelity generation") {
		t.Fatalf("expected low fidelity rejection, got %v", err)
	}
}

func TestResolveOutputDirsForBothMode(t *testing.T) {
	t.Parallel()

	hint := Hint{Bank: "bbva", Layout: "card"}
	paths := resolveOutputDirs(Config{
		Output: "testdata/public-pdfs",
		Mode:   OutputBoth,
	}, hint)

	if len(paths) != 2 {
		t.Fatalf("expected 2 output dirs, got %d", len(paths))
	}
	if paths[0] != filepath.Join("testdata", "public-pdfs", "bbva", "card") {
		t.Fatalf("unexpected public path: %q", paths[0])
	}
	if paths[1] != filepath.Join("testdata", "local-pdfs", "bbva", "card") {
		t.Fatalf("unexpected local path: %q", paths[1])
	}
}

type stubBBoxExtractor struct {
	doc TextDocument
	err error
}

func (s stubBBoxExtractor) Extract(context.Context, []byte) (TextDocument, error) {
	return s.doc, s.err
}

type stubRasterizer struct {
	doc RasterDocument
	err error
}

func (s stubRasterizer) Rasterize(context.Context, []byte) (RasterDocument, error) {
	return s.doc, s.err
}

var errRasterUnavailable = os.ErrNotExist

func textDocFromLines(lines []string) TextDocument {
	page := TextPage{
		Number: 1,
		Width:  612,
		Height: 792,
		Lines:  make([]TextLine, 0, len(lines)),
	}
	y := 60.0
	for _, line := range lines {
		page.Lines = append(page.Lines, TextLine{
			Text: line,
			XMin: 40,
			YMin: y - 12,
			XMax: 560,
			YMax: y,
		})
		y += 22
	}

	return TextDocument{
		Tool:  "stub-bbox",
		Pages: []TextPage{page},
	}
}

func mustBlankPNG(t *testing.T, width, height int) string {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{255, 255, 255, 255})
		}
	}

	path := filepath.Join(t.TempDir(), "page-0001.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create blank png: %v", err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		t.Fatalf("encode blank png: %v", err)
	}

	return path
}

func mustPDF(t *testing.T, lines []string) []byte {
	t.Helper()

	doc := fpdf.New("P", "mm", "Letter", "")
	doc.AddPage()
	doc.SetFont("Arial", "", 11)

	for _, line := range lines {
		doc.CellFormat(0, 7, line, "", 1, "", false, 0, "")
	}

	var buf bytes.Buffer
	if err := doc.Output(&buf); err != nil {
		t.Fatalf("render pdf: %v", err)
	}

	return buf.Bytes()
}

func TestSanitizeContextIsDeterministic(t *testing.T) {
	t.Parallel()

	original := edocuenta.ParseResult{
		Statement: edocuenta.Statement{
			Bank:          edocuenta.BankHSBC,
			AccountNumber: "5470749811846577",
			PeriodStart:   time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC),
			PeriodEnd:     time.Date(2025, 10, 12, 0, 0, 0, 0, time.UTC),
			Transactions: []edocuenta.Transaction{
				{
					PostedAt:    time.Date(2025, 9, 17, 0, 0, 0, 0, time.UTC),
					Description: "SERV GAS PREMIER",
					Reference:   "ABC123456789",
					Kind:        edocuenta.TransactionKindDebit,
					AmountCents: 86807,
				},
			},
		},
		ExtractedText: "HSBC MEXICO\nNUMERO DE CUENTA: 5470 7498 1184 6577",
	}

	first := newSanitizeContext(Hint{Bank: "hsbc", Layout: "card"}, original, Overrides{})
	second := newSanitizeContext(Hint{Bank: "hsbc", Layout: "card"}, original, Overrides{})

	if first.dummy.AccountNumber != second.dummy.AccountNumber {
		t.Fatalf("expected deterministic account replacement, got %q and %q", first.dummy.AccountNumber, second.dummy.AccountNumber)
	}
	if first.dummy.Transactions[0].AmountCents != second.dummy.Transactions[0].AmountCents {
		t.Fatalf("expected deterministic amount replacement")
	}
	if first.dummy.Transactions[0].Reference != second.dummy.Transactions[0].Reference {
		t.Fatalf("expected deterministic reference replacement")
	}
}
