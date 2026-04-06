package pdftext_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/go-pdf/fpdf"
	"github.com/ledgermx/mxstatementpdf/internal/pdftext"
)

func TestLedongthucExtractsText(t *testing.T) {
	t.Parallel()

	pdfBytes := mustPDF(t, []string{
		"HSBC MEXICO",
		"Cuenta: 9988776655",
		"Periodo: 01/04/2026 - 30/04/2026",
	})

	extractor := pdftext.NewLedongthuc()
	text, err := extractor.ExtractText(context.Background(), pdfBytes)
	if err != nil {
		t.Fatalf("extract text: %v", err)
	}

	if !strings.Contains(text, "HSBC") {
		t.Fatalf("expected HSBC text, got %q", text)
	}
}

func mustPDF(t *testing.T, lines []string) []byte {
	t.Helper()

	doc := fpdf.New("P", "mm", "A4", "")
	doc.AddPage()
	doc.SetFont("Arial", "", 14)

	for _, line := range lines {
		doc.CellFormat(0, 10, line, "", 1, "", false, 0, "")
	}

	var buf bytes.Buffer
	if err := doc.Output(&buf); err != nil {
		t.Fatalf("render pdf: %v", err)
	}

	return buf.Bytes()
}
