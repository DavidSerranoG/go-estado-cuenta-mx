# BBVA

Estado: implementado para estados de cuenta y estados de cuenta de tarjeta de
crédito BBVA, con validación local de PDFs reales

Nota de extracción:

- la validación local con PDFs reales se ejecuta contra los extractores ligeros por defecto
- cuando el primer parseo todavía falla porque el texto está incompleto, quien consume la librería puede habilitar Ghostscript + Tesseract OCR como extractor de rescate

Datos esperados:

- número de cuenta
- inicio y fin del periodo
- clase de cuenta (`asset` para estados de cuenta, `liability` para tarjetas de crédito)
- campos opcionales de `Summary` cuando están presentes de forma explícita en el estado de cuenta
- lista de transacciones
- dirección de la transacción (`debit` / `credit`)
- monto
- saldo corrido

Cobertura de `Summary`:

| Layout | `AccountClass` | Campos públicos de `Summary` |
| --- | --- | --- |
| estado de cuenta | `asset` | `OpeningBalanceCents`, `ClosingBalanceCents`, `TotalDebitsCents`, `TotalCreditsCents` |
| estado de cuenta de tarjeta de crédito | `liability` | `TotalDebitsCents`, `TotalCreditsCents`, `PaymentToAvoidInterestCents` |

Entradas soportadas:

- renglones sintéticos clásicos como `01/03/2026 ... ABONO 15000.00 15000.00`
- renglones reales compactos de `Detalle de Movimientos Realizados` con `OPER/LIQ`
- estados de cuenta de tarjeta BBVA con `TU PAGO REQUERIDO ESTE PERIODO` y `DESGLOSE DE MOVIMIENTOS`
- fechas cortas de transacción con abreviaturas de mes en español como `26/DIC` o `05/MAR`
- texto extraído donde fechas, encabezados y saldos pueden llegar pegados, con poco o nada de espacio
- encabezados de cuenta provenientes de `Cuenta`, `No. de Cuenta`, `Número de cuenta` o fallback derivado de CLABE

Comportamiento actual del parser:

- detecta la cuenta a partir del campo explícito primero, y después hace fallback a los últimos 10 dígitos del cuerpo de la CLABE cuando hace falta
- para tarjetas de crédito prefiere el identificador visible enmascarado de la cuenta, luego el identificador enmascarado del titular y por último el número completo de tarjeta cuando no existe identificador enmascarado
- acepta rangos de periodo con o sin `:` y con `DEL ... AL ...` o `... - ...`
- para tarjetas de crédito acepta rangos como `25-feb-2026 al 24-mar-2026`
- mapea `MONEDA NACIONAL` y marcadores similares de pesos a `MXN`
- mapea `MONEDA DÓLARES`, `MONEDA DOLARES` y marcadores similares de cuentas en dólares a `USD`
- infiere `debit` o `credit` primero a partir del saldo corrido, después con pistas en la descripción y al final con los totales del estado
- clasifica los layouts de depósito BBVA como `AccountClass=asset` y las tarjetas de crédito BBVA como `AccountClass=liability`
- para estados de cuenta expone saldos inicial y final, además de total de cargos y abonos cuando esas etiquetas están presentes
- puede corregir un monto contaminado cuando el saldo corrido deja claro cuál era el movimiento pretendido
- para tarjetas de crédito parsea la sección `CARGOS,COMPRAS Y ABONOS REGULARES (NO A MESES)`, mapea `Fecha de cargo` a `PostedAt`, deja `BalanceCents` vacío, valida los movimientos parseados contra `TOTAL CARGOS` / `TOTAL ABONOS` y expone `PAGO PARA NO GENERAR INTERESES` cuando está presente
- para tarjetas de crédito puede unir líneas de continuación OCR como `MXP ... TIPO DE CAMBIO ...` a la descripción del movimiento anterior

Límites conocidos:

- si un renglón compacto pierde tanto el saldo corrido como los totales del resumen, las filas ambiguas entre débito y crédito pueden quedar sin resolverse
- si OCR destruye los tokens de fecha corta o elimina separadores decimales de todos los montos, el parser puede detenerse con `no transactions found`
- el fallback por CLABE asume extracción estándar de una CLABE de 18 dígitos; dígitos OCR demasiado intercalados todavía pueden fallar
- el soporte de tarjeta de crédito cubre actualmente solo `CARGOS,COMPRAS Y ABONOS REGULARES (NO A MESES)`; beneficios, glosario, notas y compras a meses se ignoran
- la moneda de tarjeta de crédito actualmente se normaliza como `MXN`
- `AverageBalanceCents`, `PaymentDueDate`, `MinimumPaymentCents`, `CreditLimitCents` y `AvailableCreditCents` permanecen vacíos salvo que un futuro layout de BBVA los exponga con etiquetas estables
