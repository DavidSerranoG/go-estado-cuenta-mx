package fixturegen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-pdf/fpdf"
)

func renderHighFidelity(doc TextDocument, raster RasterDocument, overrides Overrides) ([]byte, error) {
	if len(doc.Pages) == 0 || len(raster.Pages) == 0 || len(doc.Pages) != len(raster.Pages) {
		return nil, fmt.Errorf("high fidelity render requires matching text and raster pages")
	}

	init := fpdf.InitType{
		UnitStr: "pt",
		Size: fpdf.SizeType{
			Wd: doc.Pages[0].Width,
			Ht: doc.Pages[0].Height,
		},
	}
	pdf := fpdf.NewCustom(&init)
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)

	for i, page := range doc.Pages {
		rasterPage := raster.Pages[i]
		size := fpdf.SizeType{Wd: page.Width, Ht: page.Height}
		orientation := "P"
		if size.Wd > size.Ht {
			orientation = "L"
		}
		pdf.AddPageFormat(orientation, size)
		pdf.ImageOptions(rasterPage.Path, 0, 0, size.Wd, size.Ht, false, fpdf.ImageOptions{ImageType: imageType(rasterPage.Path)}, 0, "")

		pdf.SetFillColor(255, 255, 255)
		for _, region := range overrides.Regions {
			if region.Page != 0 && region.Page != i+1 {
				continue
			}
			pdf.Rect(region.X, region.Y, region.W, region.H, "F")
		}

		pdf.SetTextColor(0, 0, 0)
		for _, line := range page.Lines {
			if line.Text == "" {
				continue
			}

			height := line.YMax - line.YMin
			if height < 8 {
				height = 8
			}

			pdf.SetFillColor(255, 255, 255)
			pdf.Rect(line.XMin-1, line.YMin-1, (line.XMax-line.XMin)+2, height+2, "F")
			pdf.SetFont("Helvetica", "", height*0.82)
			pdf.Text(line.XMin, line.YMax-1, line.Text)
		}
	}

	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, fmt.Errorf("render high fidelity pdf: %w", err)
	}
	return out.Bytes(), nil
}

func renderFallback(doc TextDocument) ([]byte, error) {
	if len(doc.Pages) == 0 {
		return nil, fmt.Errorf("fallback render requires at least one page")
	}

	init := fpdf.InitType{
		UnitStr: "pt",
		Size: fpdf.SizeType{
			Wd: doc.Pages[0].Width,
			Ht: doc.Pages[0].Height,
		},
	}
	pdf := fpdf.NewCustom(&init)
	pdf.SetMargins(24, 24, 24)
	pdf.SetAutoPageBreak(false, 24)
	pdf.SetFont("Helvetica", "", 11)

	for _, page := range doc.Pages {
		size := fpdf.SizeType{Wd: page.Width, Ht: page.Height}
		orientation := "P"
		if size.Wd > size.Ht {
			orientation = "L"
		}
		pdf.AddPageFormat(orientation, size)
		y := 36.0
		for _, line := range page.Lines {
			if line.Text == "" {
				continue
			}
			pdf.Text(24, y, line.Text)
			y += 14
			if y >= size.Ht-24 {
				break
			}
		}
	}

	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, fmt.Errorf("render fallback pdf: %w", err)
	}
	return out.Bytes(), nil
}

func imageType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "JPG"
	default:
		return "PNG"
	}
}

func writeOutputs(outputDir, name string, pdfBytes []byte, metadata Metadata, check bool) (Metadata, error) {
	if check {
		return metadata, nil
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return metadata, fmt.Errorf("create output dir: %w", err)
	}

	pdfPath := filepath.Join(outputDir, name+".pdf")
	if err := os.WriteFile(pdfPath, pdfBytes, 0o600); err != nil {
		return metadata, fmt.Errorf("write dummy pdf: %w", err)
	}

	sidecarPath := filepath.Join(outputDir, name+".json")
	metadata.OutputPath = pdfPath
	metadata.SidecarPath = sidecarPath
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return metadata, fmt.Errorf("encode sidecar metadata: %w", err)
	}
	if err := os.WriteFile(sidecarPath, data, 0o600); err != nil {
		return metadata, fmt.Errorf("write sidecar metadata: %w", err)
	}

	return metadata, nil
}
