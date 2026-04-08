# BBVA

Status: implemented for account statements and BBVA credit card statements, with local real-PDF validation

Extraction note:

- local real-PDF validation is run against the lightweight default extractors
- when the first parse still fails because text is incomplete, callers can opt into Ghostscript + tesseract OCR as a rescue extractor

Expected data:

- account number
- period start and end
- transaction list
- transaction kind (`debit` / `credit`)
- amount
- running balance

Supported inputs:

- classic synthetic rows such as `01/03/2026 ... ABONO 15000.00 15000.00`
- compact real rows from `Detalle de Movimientos Realizados` with `OPER/LIQ`
- BBVA credit card statements with `TU PAGO REQUERIDO ESTE PERIODO` and `DESGLOSE DE MOVIMIENTOS`
- short transaction dates with Spanish month abbreviations such as `26/DIC` or `05/MAR`
- extracted text where dates, headers, and balances may arrive glued with little or no whitespace
- account headers from `Cuenta`, `No. de Cuenta`, `Número de cuenta`, or CLABE-derived fallback

Current parser behavior:

- it detects the account from the explicit account field first, then falls back to the last 10 digits of the CLABE account body when needed
- for credit cards it prefers the masked visible account identifier, then the masked card holder identifier, and finally the full card number when no masked identifier is present
- it accepts period ranges with or without `:` and with `DEL ... AL ...` or `... - ...`
- for credit cards it accepts ranges like `25-feb-2026 al 24-mar-2026`
- it maps `MONEDA NACIONAL` and similar peso markers to `MXN`
- it maps `MONEDA DÓLARES`, `MONEDA DOLARES`, and similar dollar-account markers to `USD`
- it infers `debit` or `credit` from running balance first, then description hints, then statement totals
- it can repair a contaminated amount when the running balance makes the intended movement clear
- for credit cards it parses the `CARGOS,COMPRAS Y ABONOS REGULARES (NO A MESES)` section, maps `Fecha de cargo` into `PostedAt`, keeps `BalanceCents` empty, and validates parsed movements against `TOTAL CARGOS` / `TOTAL ABONOS`
- for credit cards it can join OCR continuation lines such as `MXP ... TIPO DE CAMBIO ...` onto the preceding movement description

Known limits:

- if a compact row loses both the running balance and the summary totals, ambiguous debit vs credit rows can remain unresolved
- if OCR destroys the short date tokens or removes decimal separators from all amounts, the parser may stop with `no transactions found`
- CLABE fallback assumes standard 18-digit CLABE extraction; badly interleaved OCR digits may still fail
- credit card support currently covers only `CARGOS,COMPRAS Y ABONOS REGULARES (NO A MESES)`; benefits, glossary, notes, and purchases at instalments are ignored
- credit card currency is currently normalized as `MXN`
