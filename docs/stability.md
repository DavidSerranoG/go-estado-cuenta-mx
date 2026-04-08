# Stability and Compatibility

This project is currently pre-1.0.

## What is stable enough to build on

- The root package import alias `edocuenta`
- The `supported` package as the default external entrypoint
- The public domain types `Statement`, `Transaction`, `ParseResult`, and `ParseResult.Extraction`
- The built-in parser facades in `bbva` and `hsbc`

Current public terminology:

- `Statement.AccountClass` classifies the account as `asset` or `liability`
- `Transaction.Direction` classifies each movement as `debit` or `credit`
- `Statement.Summary` exposes optional statement-level balances, totals, and payment metadata

## What may still change

- Internal package layout
- Parser heuristics and warning strings
- OCR integrations and extractor tuning
- Exact coverage by bank layout
- Additional summary fields as more layouts are supported

## Versioning policy

- Breaking changes are allowed before `v1.0.0`
- Behavior fixes for parsers may change normalized output when the current output is incorrect
- Future release notes should call out parser-level behavior changes explicitly
- Public naming cleanups, such as `Transaction.Direction`, may still happen before `v1.0.0`
