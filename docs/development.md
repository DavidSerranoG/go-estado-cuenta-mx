# Development Notes

These commands are for maintainers working on parser behavior, extraction
quality, and sanitized fixture refreshes. They are not required for normal use
of the library.

## Local Validation

Default validation:

```bash
go test ./...
go vet ./...
```

Real bank statements stay local under `.tmp/real-pdfs/` and are intentionally
excluded from the default suite.

```bash
go run ./cmd/edocuenta-eval -root .tmp/real-pdfs -format markdown
go test -tags realpdfs ./bbva -run TestParseLocalRealPDFs -count=1 -v
go test -tags realpdfs ./hsbc -run TestParseLocalRealPDFs -count=1 -v
```

If you organize your private corpus as `.tmp/real-pdfs/<bank>/<layout>/*.pdf`,
`edocuenta-eval` will summarize results by bank and layout automatically.

## Dummy Fixture Generation

`edocuenta-fixturegen` turns local real PDFs into sanitized dummy fixtures with
a searchable text layer and sidecar metadata.

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
- JSON sidecars with replacement counts, hashes, fidelity, and validation
  results

Public mode requires high-fidelity generation by default. If the host is
missing `pdftotext -bbox-layout` or a rasterizer such as `pdftocairo`, the
command fails instead of silently emitting low-fidelity public fixtures.

Optional tooling used by `fixturegen`:

| Feature | Dependency |
| --- | --- |
| layout text boxes | `pdftotext -bbox-layout` |
| raster background | `pdftocairo`, `gs`, or PDFKit via `swift` on macOS |

Override files live under `testdata/fixturegen/overrides/<bank>/<layout>.json`.
Use them for logo regions, fixed headers, or file-specific replacements that
the automatic sanitizer should not guess.

## Privacy Rules

- Never commit real bank statements.
- Never attach unredacted statements to issues or pull requests.
- Keep local-only dummy PDFs under `testdata/local-pdfs/`; that path is ignored
  by Git.
