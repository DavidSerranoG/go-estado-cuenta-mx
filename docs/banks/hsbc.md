# HSBC

Estado: implementado y endurecido con validación local de PDFs reales

Layouts soportados:

- estado de cuenta de tarjeta de crédito HSBC
- estado de cuenta HSBC Cuenta Flexible

Datos esperados:

- número de cuenta
- inicio y fin del periodo
- clase de cuenta (`asset` para Cuenta Flexible, `liability` para tarjetas de crédito)
- campos opcionales de `Summary` cuando están presentes de forma explícita en el estado de cuenta
- lista de transacciones
- dirección de la transacción (`debit` / `credit`)
- monto
- saldo corrido cuando el layout lo expone

Cobertura de `Summary`:

| Layout | `AccountClass` | Campos públicos de `Summary` |
| --- | --- | --- |
| Cuenta Flexible | `asset` | `OpeningBalanceCents`, `ClosingBalanceCents` |
| estado de cuenta de tarjeta de crédito | `liability` | ninguno por ahora |

Comportamiento actual del parser:

- acepta `NÚMERO DE CUENTA` con o sin `:` y tolera la falta de acentos en el encabezado
- los periodos de tarjeta se parsean a partir de rangos como `15-Sep-2025 al 12-Oct-2025`
- los movimientos de tarjeta pueden venir completamente compactados en una sola línea o divididos entre una línea de detalle y una línea de monto
- los periodos de Cuenta Flexible se parsean desde rangos numéricos compactos como `01102025 al 31102025`
- clasifica los layouts de Cuenta Flexible como `AccountClass=asset` y los layouts de tarjeta como `AccountClass=liability`
- para Cuenta Flexible expone `Saldo Inicial` y `Saldo Final` mediante `Statement.Summary` cuando están presentes
- los movimientos de Cuenta Flexible se infieren a partir del saldo previo contra el saldo actual, con fallback por pista en la descripción para pagos de tarjeta
- el parseo de Cuenta Flexible se detiene antes de secciones de apéndice como SPEI, CoDi, CFDI/páginas de información general para que no generen ruido de transacciones
- la validación local con PDFs reales actualmente mide el comportamiento del parser solo contra los extractores ligeros por defecto

Límites conocidos:

- los estados de tarjeta todavía asumen pares de fecha reconocibles `dd-Mon-yyyy`; si OCR rompe tanto las fechas como los marcadores de signo, pueden perderse renglones
- Cuenta Flexible todavía espera que el encabezado de la transacción comience con un día de 2 dígitos; si OCR destruye esa pista, el movimiento se omitirá
- la moneda del estado se normaliza como `MXN`; la metadata de compras en moneda extranjera se conserva solo dentro de la descripción/texto crudo del movimiento
- los estados de tarjeta HSBC todavía no exponen fechas límite de pago, pago mínimo, límites de crédito ni crédito disponible en el modelo público de `Summary`
