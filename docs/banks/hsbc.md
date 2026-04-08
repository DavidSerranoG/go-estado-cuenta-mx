# HSBC

Status: implemented and hardened with local real-PDF validation

Supported layouts:

- HSBC credit card statement
- HSBC Cuenta Flexible statement

Expected data:

- account number
- period start and end
- transaction list
- transaction kind (`debit` / `credit`)
- amount
- running balance when present in the layout

Current parser behavior:

- it accepts `NÚMERO DE CUENTA` with or without `:` and tolerates missing accents in the heading
- card periods are parsed from ranges like `15-Sep-2025 al 12-Oct-2025`
- card movements may be fully compacted on one line or split across a detail line plus an amount line
- Cuenta Flexible periods are parsed from compact numeric ranges such as `01102025 al 31102025`
- Cuenta Flexible movements are inferred from previous balance vs current balance, with a description hint fallback for card payments
- Cuenta Flexible parsing stops before appendix sections such as SPEI, CoDi, CFDI/general-information pages so they do not create transaction noise
- local real-PDF validation currently measures parser behavior against the lightweight default extractors only

Known limits:

- card statements still assume recognizable `dd-Mon-yyyy` date pairs; if OCR breaks both dates and sign markers, rows can be lost
- Cuenta Flexible still expects the transaction header to begin with a 2-digit day; if OCR destroys that cue, the movement will be skipped
- statement currency is normalized as `MXN`; foreign-currency purchase metadata is kept only inside the movement description/raw text
