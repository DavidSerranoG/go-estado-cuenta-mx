# Overrides De `fixturegen`

`edocuenta-fixturegen` carga overrides opcionales por layout desde:

```text
testdata/fixturegen/overrides/<bank>/<layout>.json
```

Claves soportadas:

```json
{
  "regions": [
    { "page": 1, "x": 0, "y": 0, "w": 120, "h": 42, "label": "wordmark" }
  ],
  "line_replacements": [
    { "match": "HSBC MEXICO", "replace": "BANCO DEMO" }
  ],
  "files": {
    "example.pdf": {
      "regions": [],
      "line_replacements": []
    }
  }
}
```

Las coordenadas usan puntos de PDF para alinearse con
`pdftotext -bbox-layout` y con el PDF renderizado de salida.
