# Fixturegen Overrides

`edocuenta-fixturegen` loads optional per-layout overrides from:

```text
testdata/fixturegen/overrides/<bank>/<layout>.json
```

Supported keys:

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

Coordinates use PDF points so they can align with `pdftotext -bbox-layout` and
the rendered output PDF.

