package fixturegen

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

type OptionalRasterizer struct {
	lookPath func(string) (string, error)
	goos     string
}

func NewOptionalRasterizer() OptionalRasterizer {
	return OptionalRasterizer{}
}

func (r OptionalRasterizer) Rasterize(ctx context.Context, pdfBytes []byte) (RasterDocument, error) {
	if tool, binary, ok := r.findTool(); ok {
		switch tool {
		case "pdftocairo":
			return runPdftocairo(ctx, binary, pdfBytes)
		case "gs":
			return runGhostscriptRaster(ctx, binary, pdfBytes)
		case "swift":
			return runSwiftPDFKitRaster(ctx, binary, pdfBytes)
		}
	}

	return RasterDocument{}, fmt.Errorf("no rasterizer available")
}

func (r OptionalRasterizer) findTool() (string, string, bool) {
	lookPath := r.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	if binary, err := lookPath("pdftocairo"); err == nil {
		return "pdftocairo", binary, true
	}
	if binary, err := lookPath("gs"); err == nil {
		return "gs", binary, true
	}
	if r.platform() == "darwin" {
		if binary, err := lookPath("swift"); err == nil {
			return "swift", binary, true
		}
	}

	return "", "", false
}

func (r OptionalRasterizer) platform() string {
	if r.goos != "" {
		return r.goos
	}
	return runtime.GOOS
}

func runPdftocairo(ctx context.Context, binary string, pdfBytes []byte) (RasterDocument, error) {
	tempDir, inputPath, err := writeTempPDF(pdfBytes, "edocuenta-fixturegen-raster-*")
	if err != nil {
		return RasterDocument{}, err
	}
	defer os.RemoveAll(tempDir)

	outputPrefix := filepath.Join(tempDir, "page")
	cmd := exec.CommandContext(ctx, binary, "-png", "-r", "144", inputPath, outputPrefix)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return RasterDocument{}, ctxErr
		}
		if stderr.Len() > 0 {
			return RasterDocument{}, fmt.Errorf("run pdftocairo: %s", strings.TrimSpace(stderr.String()))
		}
		return RasterDocument{}, fmt.Errorf("run pdftocairo: %w", err)
	}

	return loadRasterPages("pdftocairo", filepath.Join(tempDir, "page-*.png"), 144)
}

func runGhostscriptRaster(ctx context.Context, binary string, pdfBytes []byte) (RasterDocument, error) {
	tempDir, inputPath, err := writeTempPDF(pdfBytes, "edocuenta-fixturegen-gs-*")
	if err != nil {
		return RasterDocument{}, err
	}
	defer os.RemoveAll(tempDir)

	outputPattern := filepath.Join(tempDir, "page-%04d.png")
	cmd := exec.CommandContext(
		ctx,
		binary,
		"-q",
		"-dNOSAFER",
		"-dSAFER=false",
		"-dBATCH",
		"-dNOPAUSE",
		"-sDEVICE=png16m",
		"-r144",
		"-sOutputFile="+outputPattern,
		inputPath,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return RasterDocument{}, ctxErr
		}
		if stderr.Len() > 0 {
			return RasterDocument{}, fmt.Errorf("run ghostscript rasterizer: %s", strings.TrimSpace(stderr.String()))
		}
		return RasterDocument{}, fmt.Errorf("run ghostscript rasterizer: %w", err)
	}

	return loadRasterPages("ghostscript", filepath.Join(tempDir, "page-*.png"), 144)
}

func runSwiftPDFKitRaster(ctx context.Context, binary string, pdfBytes []byte) (RasterDocument, error) {
	tempDir, inputPath, err := writeTempPDF(pdfBytes, "edocuenta-fixturegen-pdfkit-*")
	if err != nil {
		return RasterDocument{}, err
	}
	defer os.RemoveAll(tempDir)

	scriptPath := filepath.Join(tempDir, "render.swift")
	if err := os.WriteFile(scriptPath, []byte(swiftPDFKitRasterizerScript), 0o600); err != nil {
		return RasterDocument{}, fmt.Errorf("write render script: %w", err)
	}

	pagesDir := filepath.Join(tempDir, "pages")
	cmd := exec.CommandContext(ctx, binary, scriptPath, inputPath, pagesDir, "144")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return RasterDocument{}, ctxErr
		}
		if stderr.Len() > 0 {
			return RasterDocument{}, fmt.Errorf("run swift pdfkit rasterizer: %s", strings.TrimSpace(stderr.String()))
		}
		return RasterDocument{}, fmt.Errorf("run swift pdfkit rasterizer: %w", err)
	}

	return loadRasterPages("pdfkit", filepath.Join(pagesDir, "page-*.png"), 144)
}

func writeTempPDF(pdfBytes []byte, pattern string) (string, string, error) {
	tempDir, err := os.MkdirTemp("", pattern)
	if err != nil {
		return "", "", fmt.Errorf("create temp dir: %w", err)
	}

	inputPath := filepath.Join(tempDir, "input.pdf")
	if err := os.WriteFile(inputPath, pdfBytes, 0o600); err != nil {
		os.RemoveAll(tempDir)
		return "", "", fmt.Errorf("write temp pdf: %w", err)
	}

	return tempDir, inputPath, nil
}

func loadRasterPages(tool, glob string, dpi float64) (RasterDocument, error) {
	files, err := filepath.Glob(glob)
	if err != nil {
		return RasterDocument{}, fmt.Errorf("glob raster pages: %w", err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		return RasterDocument{}, fmt.Errorf("no raster pages produced")
	}

	pages := make([]RasterPage, 0, len(files))
	for _, file := range files {
		width, height, err := rasterSize(file, dpi)
		if err != nil {
			return RasterDocument{}, err
		}
		pages = append(pages, RasterPage{
			Path:   file,
			Width:  width,
			Height: height,
		})
	}

	return RasterDocument{
		Pages: pages,
		Tool:  tool,
	}, nil
}

func rasterSize(path string, dpi float64) (float64, float64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, fmt.Errorf("open raster page: %w", err)
	}
	defer file.Close()

	cfg, err := png.DecodeConfig(file)
	if err != nil {
		return 0, 0, fmt.Errorf("decode raster page: %w", err)
	}

	width := float64(cfg.Width) * 72.0 / dpi
	height := float64(cfg.Height) * 72.0 / dpi
	return width, height, nil
}

const swiftPDFKitRasterizerScript = `
import AppKit
import Foundation
import PDFKit

let args = CommandLine.arguments
guard args.count == 4 else {
    fputs("usage: render.swift input.pdf output_dir dpi\n", stderr)
    exit(2)
}

let inputURL = URL(fileURLWithPath: args[1])
let outputDir = URL(fileURLWithPath: args[2], isDirectory: true)
guard let dpi = Double(args[3]) else {
    fputs("invalid dpi\n", stderr)
    exit(2)
}

guard let document = PDFDocument(url: inputURL) else {
    fputs("open pdf failed\n", stderr)
    exit(1)
}

try? FileManager.default.createDirectory(at: outputDir, withIntermediateDirectories: true)

func render(page: PDFPage, dpi: Double) -> NSBitmapImageRep? {
    let bounds = page.bounds(for: .mediaBox)
    let scale = CGFloat(dpi / 72.0)
    let width = max(Int((bounds.width * scale).rounded(.up)), 1)
    let height = max(Int((bounds.height * scale).rounded(.up)), 1)

    guard let bitmap = NSBitmapImageRep(
        bitmapDataPlanes: nil,
        pixelsWide: width,
        pixelsHigh: height,
        bitsPerSample: 8,
        samplesPerPixel: 4,
        hasAlpha: true,
        isPlanar: false,
        colorSpaceName: .deviceRGB,
        bytesPerRow: 0,
        bitsPerPixel: 0
    ) else {
        return nil
    }

    NSGraphicsContext.saveGraphicsState()
    guard let context = NSGraphicsContext(bitmapImageRep: bitmap) else {
        return nil
    }
    NSGraphicsContext.current = context
    let cg = context.cgContext
    cg.setFillColor(NSColor.white.cgColor)
    cg.fill(CGRect(x: 0, y: 0, width: CGFloat(width), height: CGFloat(height)))
    cg.scaleBy(x: scale, y: scale)
    page.draw(with: .mediaBox, to: cg)
    context.flushGraphics()
    NSGraphicsContext.restoreGraphicsState()

    return bitmap
}

for index in 0..<document.pageCount {
    guard let page = document.page(at: index), let bitmap = render(page: page, dpi: dpi) else {
        continue
    }
    let path = outputDir.appendingPathComponent(String(format: "page-%04d.png", index + 1))
    guard let data = bitmap.representation(using: .png, properties: [:]) else {
        continue
    }
    try data.write(to: path)
}
`
