# Architecture

`go-estado-cuenta-mx` keeps its public API in the root `edocuenta` package and is intentionally small and focused.

Core boundaries:

- public API in the root package
- public parser facades in `bbva/`, `hsbc/`, and convenience wiring in `supported/`
- heavy bank parsing logic in `internal/banks/`
- shared extraction and normalization helpers in `internal/`
- internal developer tooling in `internal/fixturegen/`
- no persistence or transport concerns

Flow:

1. the consumer provides PDF bytes
2. the processor runs the configured text extraction strategy
3. a bank parser is selected explicitly or by structural detection score
4. the parser returns a normalized `Statement`
5. advanced callers can inspect `ParseResult` warnings, extraction diagnostics, and extracted text

## Built-in extraction strategy

The built-in default extractor chain is intentionally small:

- `ledongthuc` Go-native extraction
- `pdftotext` fallback when the binary is installed on the host

OCR is opt-in. Callers can replace the chain with `WithExtractor(...)`, add a
rescue extractor through `WithRescueExtractor(...)`, or build their own chain
through the public extractor constructors in the root package.

When a rescue extractor is configured, the processor now evaluates all usable
text candidates from both the primary and rescue paths and keeps the best
parseable result using a fixed policy:

- parse success first
- more normalized transactions next
- higher structural detection score next
- higher text-quality score next
- stable extractor order last

## Error model

Concrete extractors report typed attempt outcomes internally:

- `failed`
- `disabled`
- `unavailable`
- `no_text`

The processor wraps those into a public `*edocuenta.TextExtractionError` with:

- a readable multi-line message
- per-extractor attempt metadata
- `errors.Is` support for high-level conditions such as:
  - Go extraction failure
  - `pdftotext` failure
  - no extractor produced usable text

This keeps diagnostics stable even as more extractors are added later.

Successful parses also surface the selected extractor and full attempt history
through `ParseResult.Extraction`.

## Real-PDF validation

Local real-PDF validation lives behind the `realpdfs` build tag so that:

- `go test ./...` stays reliable and independent from local PDFs
- `.tmp/real-pdfs` remains useful for integration-style validation
- parser and extraction progress can be tracked against a private local corpus

The `cmd/edocuenta-eval` development command reads that private corpus and emits
JSON or Markdown summaries by bank/layout, including extractor selection,
warnings, and transaction counts.

The `cmd/edocuenta-fixturegen` development command reads the same private corpus
and generates sanitized dummy fixtures plus sidecar metadata. It is intentionally
kept out of the public root API so the library stays focused on parsing.
