package hsbc_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ledgermx/mxstatementpdf"
	"github.com/ledgermx/mxstatementpdf/hsbc"
)

func TestParseLocalRealPDFs(t *testing.T) {
	pattern := filepath.Join("..", ".tmp", "real-pdfs", "hsbc", "*.pdf")
	files, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob real pdfs: %v", err)
	}
	if len(files) == 0 {
		t.Skip("no local HSBC PDFs found in .tmp/real-pdfs/hsbc")
	}

	if _, err := os.Stat(filepath.Join("..", ".tmp", "venv", "bin", "python")); err == nil {
		t.Setenv("MXSTATEMENTPDF_PYTHON", filepath.Join("..", ".tmp", "venv", "bin", "python"))
	}

	processor := statementpdf.New(
		statementpdf.WithParser(hsbc.New()),
	)

	for _, file := range files {
		file := file
		t.Run(filepath.Base(file), func(t *testing.T) {
			pdfBytes, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read pdf: %v", err)
			}

			statement, err := processor.ParsePDF(context.Background(), pdfBytes)
			if err != nil {
				t.Fatalf("parse pdf: %v", err)
			}

			if statement.Bank != "hsbc" {
				t.Fatalf("expected hsbc, got %q", statement.Bank)
			}
			if len(statement.Transactions) == 0 {
				t.Fatalf("expected at least one transaction")
			}
		})
	}
}
