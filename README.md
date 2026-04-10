# go-estado-cuenta-mx

`go-estado-cuenta-mx` is a Go library for parsing Mexican bank statement PDFs
into a normalized domain model.

- Module path: `github.com/DavidSerranoG/go-estado-cuenta-mx`
- Recommended import alias: `edocuenta`
- Recommended external entrypoint: `supported.New()`

The project is pre-1.0. Public package names are stabilizing, but parser
coverage and some pre-v1 behavior can still change as supported layouts expand.

## What This Repository Is

This repository is focused on one job: convert supported Mexican bank statement
PDFs into a clean Go data model that is easier to consume than raw PDF text.

It is intentionally not:

- a general OCR framework
- a banking integration
- a persistence layer
- a financial reconciliation system

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

For advanced callers that need warnings, extraction diagnostics, or extracted
text:

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

## Supported Banks

Coverage is layout-specific, not bank-wide.

| Bank | Supported layouts | Notes |
| --- | --- | --- |
| BBVA | account statements, credit card statements | CLABE fallback and rescue OCR supported |
| HSBC | credit card statements, Cuenta Flexible | OCR-heavy card flows supported |

Detailed notes:

- [BBVA](docs/banks/bbva.md)
- [HSBC](docs/banks/hsbc.md)
- [Supported banks and limits](docs/supported-banks.md)
- [Stability and compatibility](docs/stability.md)

## Public API

The main domain types are:

- `Statement`
- `Transaction`
- `ParseResult`

`Statement` includes:

- `Bank`
- `AccountNumber`
- `Currency`
- `PeriodStart`
- `PeriodEnd`
- `AccountClass`
- `Summary`
- `Transactions`

`AccountClass` describes the account itself:

- `asset`
- `liability`

`Transaction.Direction` describes each movement:

- `debit`
- `credit`

`Summary` is optional and only exposes values clearly present in the source
statement, such as balances, totals, or card payment metadata.

Diagnostics live in `ParseResult`, not in `Statement`.

## Extraction Strategy

Default extraction stays intentionally small and predictable:

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

## Limits and Privacy

- This project is not affiliated with BBVA, HSBC, or any other bank.
- Support is limited to the documented statement layouts. Unsupported or heavily
  degraded PDFs can fail or produce incomplete output.
- OCR is not enabled automatically and depends on host tooling when configured.
- Do not open issues or pull requests with real or unredacted bank statements.
  Share only sanitized text or sanitized fixtures.

## Development Tooling

This repository includes maintainer tooling for evaluating parsers against a
private local corpus and generating sanitized dummy fixtures. Those commands are
for development workflows, not part of the main public API.

See [Development notes](docs/development.md) for local validation, real-PDF
testing, and fixture generation.

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
