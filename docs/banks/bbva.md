# BBVA

Status: implemented MVP

Expected data:

- account number
- period start and end
- transaction list
- movement type
- amount
- running balance

Current parser assumptions:

- the statement text contains `BBVA`
- account appears as `Cuenta:`
- period appears as `Periodo: 01/03/2026 - 31/03/2026`
- transactions follow a single-line layout with `ABONO` or `CARGO`

Next step:

- validate and harden with real anonymized PDFs
