package fixturegen

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type PdftotextBBoxExtractor struct {
	lookPath func(string) (string, error)
	run      func(context.Context, string, []byte) ([]byte, error)
}

func NewPdftotextBBoxExtractor() PdftotextBBoxExtractor {
	return PdftotextBBoxExtractor{}
}

func (e PdftotextBBoxExtractor) Extract(ctx context.Context, pdfBytes []byte) (TextDocument, error) {
	lookPath := e.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	binary, err := lookPath("pdftotext")
	if err != nil {
		return TextDocument{}, fmt.Errorf("find pdftotext binary: %w", err)
	}

	run := e.run
	if run == nil {
		run = runPdftotextBBox
	}

	data, err := run(ctx, binary, pdfBytes)
	if err != nil {
		return TextDocument{}, err
	}

	doc, err := parseBBoxDocument(data)
	if err != nil {
		return TextDocument{}, err
	}
	doc.Tool = "pdftotext -bbox-layout"
	return doc, nil
}

func runPdftotextBBox(ctx context.Context, binary string, pdfBytes []byte) ([]byte, error) {
	input, err := os.CreateTemp("", "edocuenta-fixturegen-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("create temp pdf: %w", err)
	}
	defer os.Remove(input.Name())

	if _, err := input.Write(pdfBytes); err != nil {
		input.Close()
		return nil, fmt.Errorf("write temp pdf: %w", err)
	}
	if err := input.Close(); err != nil {
		return nil, fmt.Errorf("close temp pdf: %w", err)
	}

	cmd := exec.CommandContext(ctx, binary, "-bbox-layout", input.Name(), "-")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("run pdftotext -bbox-layout: %s", strings.TrimSpace(stderr.String()))
		}
		return nil, fmt.Errorf("run pdftotext -bbox-layout: %w", err)
	}

	return stdout.Bytes(), nil
}

func parseBBoxDocument(data []byte) (TextDocument, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	doc := TextDocument{}

	var currentPage *TextPage
	var currentLine *TextLine
	var currentWord strings.Builder
	inWord := false

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return TextDocument{}, fmt.Errorf("decode bbox xml: %w", err)
		}

		switch node := token.(type) {
		case xml.StartElement:
			switch node.Name.Local {
			case "page":
				page := TextPage{
					Number: parseAttrFloat(node.Attr, "number"),
					Width:  parseAttrFloat(node.Attr, "width"),
					Height: parseAttrFloat(node.Attr, "height"),
				}
				doc.Pages = append(doc.Pages, page)
				currentPage = &doc.Pages[len(doc.Pages)-1]
			case "line":
				if currentPage == nil {
					continue
				}
				line := TextLine{
					XMin: parseAttrFloat(node.Attr, "xMin"),
					YMin: parseAttrFloat(node.Attr, "yMin"),
					XMax: parseAttrFloat(node.Attr, "xMax"),
					YMax: parseAttrFloat(node.Attr, "yMax"),
				}
				currentPage.Lines = append(currentPage.Lines, line)
				currentLine = &currentPage.Lines[len(currentPage.Lines)-1]
			case "word":
				inWord = true
				currentWord.Reset()
			}
		case xml.CharData:
			if inWord {
				currentWord.Write([]byte(node))
			}
		case xml.EndElement:
			switch node.Name.Local {
			case "word":
				if currentLine != nil {
					word := strings.TrimSpace(currentWord.String())
					if word != "" {
						if currentLine.Text != "" {
							currentLine.Text += " "
						}
						currentLine.Text += word
					}
				}
				currentWord.Reset()
				inWord = false
			case "line":
				currentLine = nil
			case "page":
				currentPage = nil
			}
		}
	}

	return doc, nil
}

func parseAttrFloat(attrs []xml.Attr, key string) float64 {
	for _, attr := range attrs {
		if attr.Name.Local != key {
			continue
		}
		value, err := strconv.ParseFloat(attr.Value, 64)
		if err == nil {
			return value
		}
	}
	return 0
}
