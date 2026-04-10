# Contribuir

## Alcance

Este repositorio está enfocado en parsear PDFs de estados de cuenta bancarios
de México a un modelo de dominio normalizado en Go. Mantén las contribuciones
alineadas con ese alcance.

## Flujo de Trabajo

1. Agrega o actualiza primero los fixtures sintéticos y las pruebas.
2. Mantén los cambios de parser aislados por banco y layout cuando sea posible.
3. Evita introducir dependencias específicas del host en la ruta por defecto.
4. Ejecuta `go test ./...` antes de abrir un PR.
5. Usa `go run ./cmd/edocuenta-eval -root .tmp/real-pdfs` cuando cambies el comportamiento de extracción u OCR.
6. Usa `go run ./cmd/edocuenta-fixturegen -input .tmp/real-pdfs -output testdata -mode both` cuando actualices los fixtures dummy públicos.

Los comandos detallados para personas mantenedoras viven en
[docs/development.md](docs/development.md).

## Notas De Diseño

- `Statement` y `Transaction` son tipos limpios de dominio público.
- Las advertencias del parser y el texto extraído pertenecen a `ParseResult`.
- La selección del extractor y el historial de intentos pertenecen a `ParseResult.Extraction`.
- OCR es opt-in y debe quedar documentado de forma explícita.

## PDFs Reales

Nunca hagas commit de estados de cuenta reales. Usa `.tmp/real-pdfs/` de forma
local con el build tag `realpdfs`.

Los PDFs dummy solo locales generados pertenecen a `testdata/local-pdfs/` y
Git los ignora.
