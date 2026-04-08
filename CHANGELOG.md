# Changelog

## Unreleased

- Breaking API cleanup: renamed `Transaction.Kind` to `Transaction.Direction` and `TransactionKind` to `TransactionDirection`.
- Added `Statement.AccountClass` with accounting terminology (`asset` / `liability`).
- Added optional public `Statement.Summary` metadata for supported bank layouts.
- Rebranded the module as `github.com/DavidSerranoG/go-estado-cuenta-mx` with `edocuenta` as the official import alias.
- Introduced `ParseResult` to separate domain data from warnings and extracted text.
- Added typed public domain enums for banks, currencies, account classes, and transaction directions.
- Added the `supported` package as the recommended external entrypoint.
- Moved heavy parser logic behind thin public bank facades and made the default extraction path lightweight.
- Added extraction candidate evaluation, OCR-aware shared normalization, and `ParseResult.Extraction` diagnostics.
- Added the `cmd/edocuenta-eval` development command for local corpus benchmarking.
- Added OSS-facing docs, license, and stronger CI checks.
