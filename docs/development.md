# Notas De Desarrollo

Estos comandos son para personas mantenedoras que trabajan en el comportamiento
de los parsers, la calidad de extracción y la actualización de fixtures
sanitizados. No son necesarios para el uso normal de la librería.

## Validación Local

Validación por defecto:

```bash
go test ./...
go vet ./...
```

Los estados de cuenta bancarios reales permanecen locales en
`.tmp/real-pdfs/` y están excluidos de forma intencional del suite por
defecto.

```bash
go run ./cmd/edocuenta-eval -root .tmp/real-pdfs -format markdown
go test -tags realpdfs ./bbva -run TestParseLocalRealPDFs -count=1 -v
go test -tags realpdfs ./hsbc -run TestParseLocalRealPDFs -count=1 -v
```

Si organizas tu corpus privado como `.tmp/real-pdfs/<bank>/<layout>/*.pdf`,
`edocuenta-eval` resumirá los resultados por banco y layout automáticamente.

## Generación De Fixtures Dummy

`edocuenta-fixturegen` convierte PDFs reales locales en fixtures dummy
sanitizados con una capa de texto indexable y metadata sidecar.

Flujo recomendado:

```bash
go run ./cmd/edocuenta-fixturegen \
  -input .tmp/real-pdfs \
  -output testdata \
  -mode both \
  -branding mixed
```

Eso escribe:

- fixtures públicos en `testdata/public-pdfs/<bank>/<layout>/`
- fixtures solo locales en `testdata/local-pdfs/<bank>/<layout>/`
- sidecars JSON con conteos de reemplazo, hashes, fidelidad y resultados de validación

El modo público exige generación de alta fidelidad por defecto. Si al host le
falta `pdftotext -bbox-layout` o un rasterizador como `pdftocairo`, el comando
falla en lugar de emitir silenciosamente fixtures públicos de baja fidelidad.

Herramientas opcionales usadas por `fixturegen`:

| Función | Dependencia |
| --- | --- |
| cajas de texto del layout | `pdftotext -bbox-layout` |
| fondo rasterizado | `pdftocairo`, `gs` o PDFKit vía `swift` en macOS |

Los archivos de overrides viven bajo
`testdata/fixturegen/overrides/<bank>/<layout>.json`. Úsalos para regiones de
logotipos, encabezados fijos o reemplazos específicos por archivo que el
sanitizador automático no debería adivinar.

## Reglas De Privacidad

- Nunca hagas commit de estados de cuenta reales.
- Nunca adjuntes estados de cuenta sin redacción a issues o pull requests.
- Mantén los PDFs dummy solo locales bajo `testdata/local-pdfs/`; Git ignora esa ruta.
