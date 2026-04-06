# Architecture

`mxstatementpdf` is intentionally small and focused.

Core boundaries:

- public API in the root package
- one package per bank parser
- shared extraction and normalization helpers in `internal/`
- no persistence or transport concerns

Flow:

1. the consumer provides PDF bytes
2. the processor extracts plain text
3. a bank parser is selected explicitly or by detection
4. the parser returns a normalized `Statement`

Extraction note:

- the default extractor is a chain
- it first tries a Go-native path
- it can fall back to `python3` + `pypdf` for PDFs with hard font encodings
