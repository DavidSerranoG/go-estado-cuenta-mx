# Arquitectura

`go-estado-cuenta-mx` mantiene su API pública en el paquete raíz `edocuenta` y
es intencionalmente pequeño y enfocado.

Límites principales:

- API pública en el paquete raíz
- fachadas públicas de parser en `bbva/`, `hsbc/` y el cableado de conveniencia en `supported/`
- lógica pesada de parseo bancario en `internal/banks/`
- helpers compartidos de extracción y normalización en `internal/`
- tooling interno para desarrollo en `internal/fixturegen/`
- sin preocupaciones de persistencia ni transporte

Flujo:

1. quien consume la librería proporciona los bytes del PDF
2. el procesador ejecuta la estrategia de extracción de texto configurada
3. se selecciona un parser bancario de forma explícita o por score de detección estructural
4. el parser devuelve un `Statement` normalizado, incluyendo `AccountClass` y datos opcionales en `Summary` cuando el layout los expone con claridad
5. quienes necesitan más detalle pueden inspeccionar advertencias, diagnósticos de extracción y texto extraído mediante `ParseResult`

`Statement` es el modelo de dominio limpio. Hoy incluye:

- clasificación de la cuenta a nivel estado mediante `AccountClass`
- metadata opcional a nivel estado mediante `Summary`
- dirección de cada movimiento mediante `Transaction.Direction`

`ParseResult` sigue existiendo solo para diagnósticos. Las advertencias, el
texto extraído y la selección del extractor se mantienen fuera de `Statement`.

## Estrategia De Extracción Integrada

La cadena de extractores integrada por defecto es intencionalmente pequeña:

- extracción nativa en Go con `ledongthuc`
- fallback con `pdftotext` cuando el binario está instalado en el host

OCR es opt-in. Quien consume la librería puede reemplazar la cadena con
`WithExtractor(...)`, agregar un extractor de rescate con
`WithRescueExtractor(...)` o construir su propia cadena con los constructores
públicos del paquete raíz.

Cuando se configura un extractor de rescate, el procesador evalúa todos los
candidatos de texto utilizables tanto de la ruta primaria como de la de
rescate, y conserva el mejor resultado parseable con una política fija:

- primero el éxito del parseo
- después más transacciones normalizadas
- después un score más alto de detección estructural
- después una puntuación más alta de calidad de texto
- y al final el orden estable del extractor

## Modelo De Errores

Los extractores concretos reportan internamente resultados tipados de intento:

- `failed`
- `disabled`
- `unavailable`
- `no_text`

El procesador los envuelve en un `*edocuenta.TextExtractionError` público con:

- un mensaje legible en varias líneas
- metadata de intentos por extractor
- soporte para `errors.Is` con condiciones de alto nivel como:
  - falla de extracción en Go
  - falla de `pdftotext`
  - ningún extractor produjo texto utilizable

Eso mantiene los diagnósticos estables aunque más adelante se agreguen más
extractores.

Los parseos exitosos también exponen el extractor seleccionado y el historial
completo de intentos mediante `ParseResult.Extraction`.

## Validación Con PDFs Reales

La validación local con PDFs reales vive detrás del build tag `realpdfs` para
que:

- `go test ./...` se mantenga confiable e independiente de PDFs locales
- `.tmp/real-pdfs` siga siendo útil para validación de tipo integración
- el progreso de parsers y extracción pueda medirse contra un corpus privado local

El comando de desarrollo `cmd/edocuenta-eval` lee ese corpus privado y emite
resúmenes en JSON o Markdown por banco/layout, incluyendo selección de
extractor, advertencias y conteo de transacciones.

El comando de desarrollo `cmd/edocuenta-fixturegen` lee el mismo corpus privado
y genera fixtures dummy sanitizados más metadatos sidecar. Se mantiene
intencionalmente fuera de la API pública raíz para que la librería siga
enfocada en el parseo.
