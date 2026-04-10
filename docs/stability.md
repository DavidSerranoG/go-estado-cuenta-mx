# Estabilidad Y Compatibilidad

Este proyecto está actualmente en pre-1.0.

## Qué Ya Es Lo Bastante Estable Para Usarlo Como Base

- el alias de importación `edocuenta` del paquete raíz
- el paquete `supported` como punto de entrada externo por defecto
- los tipos públicos de dominio `Statement`, `Transaction`, `ParseResult` y `ParseResult.Extraction`
- las fachadas integradas de parser en `bbva` y `hsbc`

Terminología pública actual:

- `Statement.AccountClass` clasifica la cuenta como `asset` o `liability`
- `Transaction.Direction` clasifica cada movimiento como `debit` o `credit`
- `Statement.Summary` expone saldos, totales y metadata de pagos a nivel estado cuando son opcionales y están disponibles

## Qué Todavía Puede Cambiar

- el layout interno de paquetes
- las heurísticas del parser y los textos de advertencia
- las integraciones de OCR y el tuning de extractores
- la cobertura exacta por banco y layout
- campos adicionales de `Summary` conforme se soporten más layouts

## Política De Versionado

- los cambios incompatibles están permitidos antes de `v1.0.0`
- las correcciones de comportamiento en parsers pueden cambiar la salida normalizada cuando la salida actual es incorrecta
- las notas de futuras versiones deberían señalar explícitamente los cambios de comportamiento a nivel parser
- todavía pueden ocurrir limpiezas de nombres públicos, como `Transaction.Direction`, antes de `v1.0.0`
