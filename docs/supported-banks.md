# Bancos Soportados Y Límites Conocidos

## BBVA

- Layouts soportados: estados de cuenta, estados de cuenta de tarjeta de crédito
- Aspectos destacados: fallback por CLABE, parseo compacto de renglones y unión de continuaciones OCR en estados de tarjeta
- Límites conocidos: el OCR muy degradado todavía puede perder fechas cortas o totales

Consulta las [notas bancarias de BBVA](banks/bbva.md).

## HSBC

- Layouts soportados: estados de cuenta de tarjeta de crédito, Cuenta Flexible
- Aspectos destacados: renglones compactos de tarjeta, renglones OCR divididos y corte de apéndices en Cuenta Flexible
- Límites conocidos: las pistas de fecha dañadas por OCR todavía pueden hacer que se pierdan movimientos

Consulta las [notas bancarias de HSBC](banks/hsbc.md).

## Cobertura Pública De `Summary`

| Banco | Layout | `AccountClass` | Campos de `Summary` |
| --- | --- | --- | --- |
| BBVA | estado de cuenta | `asset` | `OpeningBalanceCents`, `ClosingBalanceCents`, `TotalDebitsCents`, `TotalCreditsCents` |
| BBVA | estado de cuenta de tarjeta de crédito | `liability` | `TotalDebitsCents`, `TotalCreditsCents`, `PaymentToAvoidInterestCents` |
| HSBC | Cuenta Flexible | `asset` | `OpeningBalanceCents`, `ClosingBalanceCents` |
| HSBC | estado de cuenta de tarjeta de crédito | `liability` | ninguno por ahora |

`AccountClass` describe la cuenta del estado en términos contables.
`Transaction.Direction` describe la dirección de débito o crédito de cada
movimiento. Están relacionados, pero son piezas intencionalmente distintas de
la API pública.

## Consideraciones Sobre Extracción

- el parseo por defecto no habilita OCR automáticamente
- `pdftotext`, Vision OCR y Tesseract son dependencias opcionales del host
- la cobertura real se mide mejor con fixtures sintéticos más validación local con PDFs reales
