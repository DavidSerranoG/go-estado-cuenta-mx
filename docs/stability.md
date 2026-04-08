# Stability and Compatibility

This project is currently pre-1.0.

## What is stable enough to build on

- The root package import alias `edocuenta`
- The `supported` package as the default external entrypoint
- The public domain types `Statement`, `Transaction`, `ParseResult`, and `ParseResult.Extraction`
- The built-in parser facades in `bbva` and `hsbc`

## What may still change

- Internal package layout
- Parser heuristics and warning strings
- OCR integrations and extractor tuning
- Exact coverage by bank layout

## Versioning policy

- Breaking changes are allowed before `v1.0.0`
- Behavior fixes for parsers may change normalized output when the current output is incorrect
- Future release notes should call out parser-level behavior changes explicitly
