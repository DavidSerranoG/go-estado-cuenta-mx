# go-estado-cuenta-mx

`go-estado-cuenta-mx` es una biblioteca en Go para parsear PDFs de estados de
cuenta bancarios de México a un modelo de dominio normalizado.

- Ruta del módulo: `github.com/DavidSerranoG/go-estado-cuenta-mx`
- Alias de importación recomendado: `edocuenta`
- Punto de entrada externo recomendado: `supported.New()`

El proyecto sigue en pre-1.0. Los nombres públicos de los paquetes se están
estabilizando, pero la cobertura de parsers y algunos comportamientos previos a
v1 todavía pueden cambiar conforme se amplíen los layouts soportados.

## Qué Hace Este Repositorio

Este repositorio está enfocado en una sola tarea: convertir PDFs soportados de
estados de cuenta bancarios de México en un modelo de datos limpio en Go, más
fácil de consumir que el texto crudo del PDF.

Intencionalmente no pretende ser:

- un framework general de OCR
- una integración bancaria
- una capa de persistencia
- un sistema de conciliación financiera

## Instalación

```bash
go get github.com/DavidSerranoG/go-estado-cuenta-mx
```

## Inicio Rápido

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/DavidSerranoG/go-estado-cuenta-mx/supported"
)

func main() {
	pdfBytes, err := os.ReadFile("statement.pdf")
	if err != nil {
		log.Fatal(err)
	}

	processor := supported.New()

	statement, err := processor.ParsePDF(context.Background(), pdfBytes)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("bank=%s account=%s tx=%d", statement.Bank, statement.AccountNumber, len(statement.Transactions))
}
```

Para casos más avanzados donde necesites advertencias, diagnósticos de
extracción o el texto extraído:

```go
result, err := supported.New().ParsePDFResult(ctx, pdfBytes)
if err != nil {
	log.Fatal(err)
}

_ = result.Statement
_ = result.Warnings
_ = result.Extraction
_ = result.Diagnostics
_ = result.ExtractedText
```

## Bancos Soportados

La cobertura depende del layout, no del banco completo.

| Banco | Layouts soportados | Notas |
| --- | --- | --- |
| BBVA | estados de cuenta, estados de cuenta de tarjeta de crédito | Soporta fallback por CLABE y OCR de rescate |
| HSBC | estados de cuenta de tarjeta de crédito, Cuenta Flexible | Soporta flujos de tarjeta con OCR intensivo |

Notas detalladas:

- [BBVA](docs/banks/bbva.md)
- [HSBC](docs/banks/hsbc.md)
- [Bancos soportados y límites](docs/supported-banks.md)
- [Estabilidad y compatibilidad](docs/stability.md)

## API Pública

Los tipos de dominio principales son:

- `Statement`
- `Transaction`
- `ParseResult`

`Statement` incluye:

- `Bank`
- `AccountNumber`
- `Currency`
- `PeriodStart`
- `PeriodEnd`
- `AccountClass`
- `Summary`
- `Transactions`

`AccountClass` describe la cuenta en sí:

- `asset`
- `liability`

`Transaction.Direction` describe cada movimiento:

- `debit`
- `credit`

`Summary` es opcional y solo expone valores claramente presentes en el estado
de cuenta de origen, como saldos, totales o metadatos de pagos de tarjeta.

Los diagnósticos viven en `ParseResult`, no en `Statement`.

`ParseResult.Diagnostics` expone además:

- `SelectedParser`
- `Layout`
- `DetectionScore`
- `Confidence`
- `Issues`

## Estrategia de Extracción

La extracción por defecto se mantiene intencionalmente pequeña y predecible:

- `ledongthuc` primero
- `pdftotext` después, cuando el binario está disponible en el host

OCR se habilita de forma explícita. Para activar OCR de rescate:

```go
import (
	edocuenta "github.com/DavidSerranoG/go-estado-cuenta-mx"
	"github.com/DavidSerranoG/go-estado-cuenta-mx/supported"
)

processor := supported.New(
	edocuenta.WithRescueExtractor(edocuenta.NewTesseractExtractor()),
)
```

Constructores públicos de extractores disponibles:

- `NewLedongthucExtractor()`
- `NewPdftotextExtractor()`
- `NewVisionExtractor()`
- `NewTesseractExtractor()`
- `NewTextExtractorChain(...)`

## Dependencias Opcionales

| Función | Dependencia | Obligatoria por defecto |
| --- | --- | --- |
| Extracción de texto nativa en Go | Solo Go | sí |
| Fallback con `pdftotext` | Binario `pdftotext` | no |
| OCR con Vision | `swift` en macOS | no |
| OCR de rescate con Tesseract | `tesseract` y `gs` | no |

## Límites y Privacidad

- Este proyecto no está afiliado con BBVA, HSBC ni con ningún otro banco.
- El soporte está limitado a los layouts documentados. Los PDFs no soportados o
  muy degradados pueden fallar o producir salidas incompletas.
- OCR no se habilita automáticamente y depende de herramientas del host cuando
  se configura.
- No abras issues ni pull requests con estados de cuenta reales o sin redactar.
  Comparte solo texto sanitizado o fixtures sanitizados.

## Herramientas de Desarrollo

Este repositorio incluye herramientas para personas mantenedoras que evalúan
parsers contra un corpus privado local y generan fixtures dummy sanitizados.
Esos comandos son para flujos de desarrollo, no forman parte de la API pública
principal.

Consulta las [Notas de desarrollo](docs/development.md) para validación local,
pruebas con PDFs reales y generación de fixtures.

## Estructura Del Proyecto

```text
go-estado-cuenta-mx/
  bbva/
  hsbc/
  supported/
  internal/banks/
  internal/pdftext/
  internal/normalize/
  docs/
  examples/
```

## Contribución Y Seguridad

- [Contribuir](CONTRIBUTING.md)
- [Seguridad](SECURITY.md)
- [Registro de cambios](CHANGELOG.md)
