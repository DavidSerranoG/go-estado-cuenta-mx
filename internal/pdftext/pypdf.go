package pdftext

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

const pyPDFScript = `
import re
import sys
from pypdf import PdfReader

reader = PdfReader(sys.argv[1])
text = "\n".join((page.extract_text() or "") for page in reader.pages)
text = re.sub(r"/EX(\d{2,3})000", lambda m: chr(int(m.group(1))), text)
text = text.replace("/", "")
sys.stdout.write(text)
`

// PyPDF uses python3 + pypdf as a fallback extractor for difficult PDFs.
type PyPDF struct {
	pythonBinary string
}

// NewPyPDF returns a new PyPDF extractor. If pythonBinary is empty, it falls back to
// MXSTATEMENTPDF_PYTHON or python3.
func NewPyPDF(pythonBinary string) PyPDF {
	return PyPDF{pythonBinary: pythonBinary}
}

// ExtractText converts a PDF into plain text using pypdf.
func (p PyPDF) ExtractText(ctx context.Context, pdfBytes []byte) (string, error) {
	tmpFile, err := os.CreateTemp("", "mxstatementpdf-pypdf-*.pdf")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := io.Copy(tmpFile, bytes.NewReader(pdfBytes)); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("write temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("close temp file: %w", err)
	}

	cmd := exec.CommandContext(ctx, p.python(), "-", tmpFile.Name())
	cmd.Stdin = strings.NewReader(pyPDFScript)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("pypdf extractor: %s", strings.TrimSpace(stderr.String()))
		}
		return "", fmt.Errorf("pypdf extractor: %w", err)
	}

	return stdout.String(), nil
}

func (p PyPDF) python() string {
	if p.pythonBinary != "" {
		return p.pythonBinary
	}
	if env := strings.TrimSpace(os.Getenv("MXSTATEMENTPDF_PYTHON")); env != "" {
		return env
	}
	return "python3"
}
