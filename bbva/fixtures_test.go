//go:build realpdfs
// +build realpdfs

package bbva_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	edocuenta "github.com/DavidSerranoG/go-estado-cuenta-mx"
	"github.com/DavidSerranoG/go-estado-cuenta-mx/bbva"
)

func TestParseLocalRealPDFs(t *testing.T) {
	pattern := filepath.Join("..", ".tmp", "real-pdfs", "bbva", "*.pdf")
	files, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob real pdfs: %v", err)
	}
	if len(files) == 0 {
		t.Skip("no local BBVA PDFs found in .tmp/real-pdfs/bbva")
	}

	processor := edocuenta.New(
		edocuenta.WithParser(bbva.New()),
	)

	passCount := 0

	for _, file := range files {
		file := file
		t.Run(filepath.Base(file), func(t *testing.T) {
			pdfBytes, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read pdf: %v", err)
			}

			statement, err := processor.ParsePDF(context.Background(), pdfBytes)
			if err != nil {
				t.Logf("FAIL %s: %v", filepath.Base(file), err)
				return
			}

			passCount++
			t.Logf("PASS %s: bank=%s tx=%d", filepath.Base(file), statement.Bank, len(statement.Transactions))
		})
	}

	t.Logf("summary pass=%d fail=%d", passCount, len(files)-passCount)
}
