package pdftext_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/DavidSerranoG/go-estado-cuenta-mx/internal/pdftext"
	"github.com/go-pdf/fpdf"
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

func TestLedongthucHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := pdftext.NewLedongthuc().ExtractText(ctx, mustPDF(t, []string{"BBVA"}))
	if err == nil {
		t.Fatal("expected context error")
	}
	if !strings.Contains(err.Error(), context.Canceled.Error()) {
		t.Fatalf("expected canceled context in error, got %v", err)
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
