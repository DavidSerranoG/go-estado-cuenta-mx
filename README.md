# mxstatementpdf

`mxstatementpdf` is a standalone Go package for parsing Mexican bank statement PDFs into a neutral data structure that other applications can persist or transform.

Current scope:

- BBVA support
- HSBC credit card statements
- HSBC Cuenta Flexible statements
- PDF text extraction
- optional Python `pypdf` fallback for difficult PDFs
- bank detection
- normalized statement output

Out of scope:

- databases
- HTTP handlers
- queues or workers
- storage adapters
- business workflows from consuming applications

## Installation

```bash
go get github.com/ledgermx/mxstatementpdf
```

## Usage

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/ledgermx/mxstatementpdf"
	"github.com/ledgermx/mxstatementpdf/bbva"
	"github.com/ledgermx/mxstatementpdf/hsbc"
)

func main() {
	pdfBytes, err := os.ReadFile("statement.pdf")
	if err != nil {
		log.Fatal(err)
	}

	processor := statementpdf.New(
		statementpdf.WithParser(bbva.New()),
		statementpdf.WithParser(hsbc.New()),
	)

	statement, err := processor.ParsePDF(context.Background(), pdfBytes)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("bank=%s account=%s tx=%d", statement.Bank, statement.AccountNumber, len(statement.Transactions))
}
```

## Project layout

```text
mxstatementpdf/
  hsbc/
  bbva/
  internal/pdftext/
  internal/normalize/
  internal/registry/
  docs/
  examples/
  testdata/
```

## Local real PDFs

Use `.tmp/real-pdfs/` for local test files. That folder is ignored and must never be committed.

## Extractor strategy

By default the processor tries:

1. a Go-native extractor
2. a Python `pypdf` fallback if `python3` and `pypdf` are available

Some real HSBC PDFs require the fallback because they use font encodings that common Go PDF text extractors do not decode reliably yet.

If needed, you can point the fallback to a specific Python binary with `MXSTATEMENTPDF_PYTHON=/path/to/python`.
