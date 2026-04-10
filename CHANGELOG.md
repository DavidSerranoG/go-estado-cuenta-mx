# Registro De Cambios

## Sin Publicar

- Limpieza incompatible de la API: se renombró `Transaction.Kind` a `Transaction.Direction` y `TransactionKind` a `TransactionDirection`.
- Se agregó `Statement.AccountClass` con terminología contable (`asset` / `liability`).
- Se agregó metadata pública opcional en `Statement.Summary` para layouts bancarios soportados.
- Se renombró el módulo a `github.com/DavidSerranoG/go-estado-cuenta-mx`, con `edocuenta` como alias oficial de importación.
- Se introdujo `ParseResult` para separar los datos de dominio de las advertencias y del texto extraído.
- Se agregaron enums tipados de dominio público para bancos, monedas, clases de cuenta y direcciones de transacción.
- Se agregó el paquete `supported` como punto de entrada externo recomendado.
- Se movió la lógica pesada de parsers detrás de fachadas públicas delgadas por banco y se dejó la ruta de extracción por defecto más ligera.
- Se agregó evaluación de candidatos de extracción, normalización compartida consciente de OCR y diagnósticos en `ParseResult.Extraction`.
- Se agregó el comando de desarrollo `cmd/edocuenta-eval` para benchmarking sobre el corpus local.
- Se agregaron documentación orientada a OSS, licencia y checks de CI más estrictos.
