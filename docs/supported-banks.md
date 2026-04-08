# Supported Banks and Known Limits

## BBVA

- Supported layouts: account statements, credit card statements
- Highlights: CLABE fallback, compact row parsing, OCR continuation stitching for card statements
- Known limits: highly degraded OCR can still lose short dates or totals

See [BBVA bank notes](banks/bbva.md).

## HSBC

- Supported layouts: credit card statements, Cuenta Flexible
- Highlights: compact card rows, split OCR rows, appendix cutoff for Cuenta Flexible
- Known limits: OCR-damaged date cues can still drop movements

See [HSBC bank notes](banks/hsbc.md).

## Extraction caveats

- Default parsing does not enable OCR automatically
- `pdftotext`, Vision OCR, and Tesseract are optional host dependencies
- Real-world coverage is best measured with synthetic fixtures plus local real-PDF validation
