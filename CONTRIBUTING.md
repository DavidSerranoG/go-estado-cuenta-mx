# Contributing

## Scope

This repository focuses on parsing Mexican bank statement PDFs into a normalized Go domain model. Keep new contributions aligned with that scope.

## Workflow

1. Add or update synthetic fixtures and tests first.
2. Keep parser changes isolated by bank and layout when possible.
3. Avoid introducing host-specific dependencies into the default path.
4. Run `go test ./...` before opening a PR.
5. Use `go run ./cmd/edocuenta-eval -root .tmp/real-pdfs` when you change extraction or OCR behavior.
6. Use `go run ./cmd/edocuenta-fixturegen -input .tmp/real-pdfs -output testdata -mode both` when you refresh public dummy fixtures.

Detailed maintainer commands live in [docs/development.md](docs/development.md).

## Design Notes

- `Statement` and `Transaction` are clean public domain types.
- Parser warnings and extracted text belong in `ParseResult`.
- Extractor selection and attempt history belong in `ParseResult.Extraction`.
- OCR is opt-in and should stay explicitly documented.

## Real PDFs

Never commit real bank statements. Use `.tmp/real-pdfs/` locally with the `realpdfs` build tag.

Generated local-only dummy PDFs belong under `testdata/local-pdfs/` and are ignored by Git.
