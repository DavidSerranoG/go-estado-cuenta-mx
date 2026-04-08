# go-estado-cuenta-mx

`go-estado-cuenta-mx` is a Go library for parsing Mexican bank statement PDFs into a normalized domain model.

- Module path: `github.com/DavidSerranoG/go-estado-cuenta-mx`
- Recommended import alias: `edocuenta`
- Recommended external entrypoint: `supported.New()`

The project is still pre-1.0 and intentionally allows breaking changes while the OSS API is being cleaned up.

## Installation

```bash
go get github.com/DavidSerranoG/go-estado-cuenta-mx
```

## Quick Start

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/DavidSerranoG/go-estado-cuenta-mx/supported"
)

func main() {
	pdfBytes, err := os.ReadFile("statement.pdf")
	if err != nil {
		log.Fatal(err)
	}

	processor := supported.New()

	statement, err := processor.ParsePDF(context.Background(), pdfBytes)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("bank=%s account=%s tx=%d", statement.Bank, statement.AccountNumber, len(statement.Transactions))
}
```

For advanced callers that need warnings and extracted text:

```go
result, err := supported.New().ParsePDFResult(ctx, pdfBytes)
if err != nil {
	log.Fatal(err)
}

_ = result.Statement
_ = result.Warnings
_ = result.Extraction
_ = result.ExtractedText
```

Example output shape:

```json
{
  "Statement": {
    "Bank": "hsbc",
    "AccountNumber": "5470749811846577",
    "Currency": "MXN",
    "PeriodStart": "2025-09-15T00:00:00Z",
    "PeriodEnd": "2025-10-12T00:00:00Z",
    "Transactions": [
      {
        "PostedAt": "2025-09-17T00:00:00Z",
        "Description": "SU PAGO GRACIAS",
        "Reference": "",
        "Kind": "credit",
        "AmountCents": 2500000,
        "BalanceCents": null
      }
    ]
  },
  "Warnings": [],
  "Extraction": {
    "SelectedExtractor": "ledongthuc",
    "UsedRescue": false,
    "Attempts": [
      {
        "Extractor": "ledongthuc",
        "Status": "succeeded",
        "Message": "extractor produced usable text"
      }
    ]
  },
  "ExtractedText": "..."
}
```

## Supported Banks

| Bank | Layouts | Notes |
| --- | --- | --- |
| BBVA | account statements, credit card statements | CLABE fallback and OCR rescue supported |
| HSBC | credit card statements, Cuenta Flexible | OCR-heavy card flows supported |

Detailed notes:

- [BBVA](docs/banks/bbva.md)
- [HSBC](docs/banks/hsbc.md)
- [Supported banks and limits](docs/supported-banks.md)
- [Stability and compatibility](docs/stability.md)

## Public API

`Statement` is the clean domain model:

- `Bank`
- `AccountNumber`
- `Currency`
- `PeriodStart`
- `PeriodEnd`
- `Transactions`

`Transaction` uses normalized kinds:

- `debit`
- `credit`

Diagnostics live in `ParseResult`, not in `Statement`.

`ParseResult.Extraction` tells you which extractor candidate won, whether a rescue
extractor was ultimately selected, and which attempts were made along the way.

## Extraction Strategy

Default extraction is intentionally lightweight and predictable:

- `ledongthuc` first
- `pdftotext` second when available on the host

OCR is opt-in. To enable rescue OCR:

```go
import (
	edocuenta "github.com/DavidSerranoG/go-estado-cuenta-mx"
	"github.com/DavidSerranoG/go-estado-cuenta-mx/supported"
)

processor := supported.New(
	edocuenta.WithRescueExtractor(edocuenta.NewTesseractExtractor()),
)
```

Available public extractor constructors:

- `NewLedongthucExtractor()`
- `NewPdftotextExtractor()`
- `NewVisionExtractor()`
- `NewTesseractExtractor()`
- `NewTextExtractorChain(...)`

## Optional Dependencies

| Feature | Dependency | Required by default |
| --- | --- | --- |
| Go-native text extraction | Go only | yes |
| `pdftotext` fallback | `pdftotext` binary | no |
| Vision OCR | `swift` on macOS | no |
| Tesseract OCR rescue | `tesseract` and `gs` | no |

## Local Validation

Real PDFs stay local under `.tmp/real-pdfs/` and are not part of the default suite.

```bash
go test ./...
go run ./cmd/edocuenta-eval -root .tmp/real-pdfs -format markdown
go test -tags realpdfs ./bbva -run TestParseLocalRealPDFs -count=1 -v
go test -tags realpdfs ./hsbc -run TestParseLocalRealPDFs -count=1 -v
```

If you organize your private corpus as `.tmp/real-pdfs/<bank>/<layout>/*.pdf`,
`edocuenta-eval` will summarize results by bank and layout automatically.

## Dummy Fixture Generation

`edocuenta-fixturegen` turns local real PDFs into sanitized dummy fixtures with a
searchable text layer and sidecar metadata.

Recommended flow:

```bash
go run ./cmd/edocuenta-fixturegen \
  -input .tmp/real-pdfs \
  -output testdata \
  -mode both \
  -branding mixed
```

This writes:

- public fixtures under `testdata/public-pdfs/<bank>/<layout>/`
- local-only fixtures under `testdata/local-pdfs/<bank>/<layout>/`
- JSON sidecars with replacement counts, hashes, fidelity, and validation results

Public mode requires high-fidelity generation by default. If the host is missing
`pdftotext -bbox-layout` or a rasterizer such as `pdftocairo`, the command fails
instead of silently emitting low-fidelity public fixtures.

Optional tooling used by `fixturegen`:

| Feature | Dependency |
| --- | --- |
| layout text boxes | `pdftotext -bbox-layout` |
| raster background | `pdftocairo`, `gs`, or PDFKit via `swift` on macOS |

Override files live under `testdata/fixturegen/overrides/<bank>/<layout>.json`.
Use them for logo regions, fixed headers, or file-specific replacements that the
automatic sanitizer should not guess.

## Project Layout

```text
go-estado-cuenta-mx/
  bbva/
  hsbc/
  supported/
  internal/banks/
  internal/pdftext/
  internal/normalize/
  docs/
  examples/
```

## Contributing and Security

- [Contributing](CONTRIBUTING.md)
- [Security](SECURITY.md)
- [Changelog](CHANGELOG.md)
