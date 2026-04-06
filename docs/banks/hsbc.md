# HSBC

Status: implemented MVP

Supported layouts:

- HSBC credit card statement
- HSBC Cuenta Flexible statement

Expected data:

- account number
- period start and end
- transaction list
- movement type
- amount
- running balance when present in the layout

Current parser assumptions:

- the statement text contains `HSBC`
- card statements expose `NÚMERO DE CUENTA` and period like `15-Sep-2025 al 12-Oct-2025`
- Cuenta Flexible statements expose period like `01102025 al 31102025`
- card transactions may span one or two lines
- Cuenta Flexible transactions use the running balance to infer whether each movement is cargo or abono
- the real PDFs may require the Python fallback extractor

Next step:

- validate and harden with real anonymized PDFs
