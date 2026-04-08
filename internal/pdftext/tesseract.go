package pdftext

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Tesseract rasterizes PDF pages with Ghostscript and runs tesseract OCR.
type Tesseract struct {
	lookPath              func(string) (string, error)
	rasterizeGhostscript  func(context.Context, string, []byte) (string, []string, error)
	rasterizeDarwinPDFKit func(context.Context, string, []byte) (string, []string, error)
	ocrPages              func(context.Context, string, []string) (string, error)
	goos                  string
}

// NewTesseract returns the optional tesseract-based OCR extractor.
func NewTesseract() Tesseract {
	return Tesseract{}
}

// Name identifies the extractor in chain reports.
func (Tesseract) Name() string {
	return "tesseract"
}

// ExtractText converts a PDF into plain text using Ghostscript and tesseract.
func (t Tesseract) ExtractText(ctx context.Context, pdfBytes []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", newExtractorError("tesseract", AttemptCodeFailed, err)
	}

	lookPath := t.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	tesseract, err := lookPath("tesseract")
	if err != nil {
		return "", newExtractorError("tesseract", AttemptCodeUnavailable, fmt.Errorf("find tesseract binary: %w", err))
	}

	tempDir, pages, err := t.rasterizePages(ctx, pdfBytes)
	if err != nil {
		var unavailable *rasterizerUnavailableError
		if errors.As(err, &unavailable) {
			return "", newExtractorError("tesseract", AttemptCodeUnavailable, unavailable.Err)
		}
		return "", newExtractorError("tesseract", AttemptCodeFailed, err)
	}
	if tempDir != "" {
		defer os.RemoveAll(tempDir)
	}

	ocrPages := t.ocrPages
	if ocrPages == nil {
		ocrPages = runTesseractOCR
	}

	text, err := ocrPages(ctx, tesseract, pages)
	if err != nil {
		return "", newExtractorError("tesseract", AttemptCodeFailed, err)
	}

	trimmed := strings.TrimSpace(strings.ReplaceAll(text, "\x00", ""))
	if trimmed == "" {
		return "", nil
	}

	return trimmed, nil
}

func (t Tesseract) rasterizePages(ctx context.Context, pdfBytes []byte) (string, []string, error) {
	lookPath := t.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	var ghostscriptErr error
	if binary, err := lookPath("gs"); err == nil {
		rasterize := t.rasterizeGhostscript
		if rasterize == nil {
			rasterize = rasterizePDFPagesWithGhostscript
		}

		tempDir, pages, err := rasterize(ctx, binary, pdfBytes)
		if err == nil {
			return tempDir, pages, nil
		}
		ghostscriptErr = fmt.Errorf("run gs: %w", err)
	} else {
		ghostscriptErr = &rasterizerUnavailableError{Err: fmt.Errorf("find gs binary: %w", err)}
	}

	if t.platform() == "darwin" {
		var swiftErr error
		if binary, err := lookPath("swift"); err == nil {
			rasterize := t.rasterizeDarwinPDFKit
			if rasterize == nil {
				rasterize = rasterizePDFPagesWithSwiftPDFKit
			}

			tempDir, pages, err := rasterize(ctx, binary, pdfBytes)
			if err == nil {
				return tempDir, pages, nil
			}
			swiftErr = fmt.Errorf("run swift pdfkit rasterizer: %w", err)
		} else {
			swiftErr = &rasterizerUnavailableError{Err: fmt.Errorf("find swift binary: %w", err)}
		}

		if isUnavailableRasterizerError(ghostscriptErr) && isUnavailableRasterizerError(swiftErr) {
			return "", nil, &rasterizerUnavailableError{
				Err: errors.Join(unwrapRasterizerError(ghostscriptErr), unwrapRasterizerError(swiftErr)),
			}
		}
		if !isUnavailableRasterizerError(swiftErr) {
			return "", nil, swiftErr
		}
	}

	return "", nil, ghostscriptErr
}

func (t Tesseract) platform() string {
	if t.goos != "" {
		return t.goos
	}
	return runtime.GOOS
}

func rasterizePDFPagesWithGhostscript(ctx context.Context, binary string, pdfBytes []byte) (string, []string, error) {
	tempDir, err := os.MkdirTemp("", "edocuenta-tesseract-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temp dir: %w", err)
	}

	inputPath := filepath.Join(tempDir, "input.pdf")
	if err := os.WriteFile(inputPath, pdfBytes, 0o600); err != nil {
		os.RemoveAll(tempDir)
		return "", nil, fmt.Errorf("write temp pdf: %w", err)
	}

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
		"-r220",
		"-sOutputFile="+outputPattern,
		inputPath,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tempDir)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", nil, ctxErr
		}
		if stderr.Len() > 0 {
			return "", nil, fmt.Errorf("%s", strings.TrimSpace(stderr.String()))
		}
		return "", nil, err
	}

	pages, err := filepath.Glob(filepath.Join(tempDir, "page-*.png"))
	if err != nil {
		os.RemoveAll(tempDir)
		return "", nil, fmt.Errorf("glob rasterized pages: %w", err)
	}
	sort.Strings(pages)

	return tempDir, pages, nil
}

func rasterizePDFPagesWithSwiftPDFKit(ctx context.Context, binary string, pdfBytes []byte) (string, []string, error) {
	tempDir, err := os.MkdirTemp("", "edocuenta-pdfkit-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temp dir: %w", err)
	}

	inputPath := filepath.Join(tempDir, "input.pdf")
	if err := os.WriteFile(inputPath, pdfBytes, 0o600); err != nil {
		os.RemoveAll(tempDir)
		return "", nil, fmt.Errorf("write temp pdf: %w", err)
	}

	scriptPath := filepath.Join(tempDir, "render.swift")
	if err := os.WriteFile(scriptPath, []byte(swiftPDFKitRasterizerScript), 0o600); err != nil {
		os.RemoveAll(tempDir)
		return "", nil, fmt.Errorf("write pdfkit script: %w", err)
	}

	pagesDir := filepath.Join(tempDir, "pages")
	cmd := exec.CommandContext(ctx, binary, scriptPath, inputPath, pagesDir, "300")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tempDir)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", nil, ctxErr
		}
		if stderr.Len() > 0 {
			return "", nil, fmt.Errorf("%s", strings.TrimSpace(stderr.String()))
		}
		return "", nil, err
	}

	pages, err := filepath.Glob(filepath.Join(pagesDir, "page-*.png"))
	if err != nil {
		os.RemoveAll(tempDir)
		return "", nil, fmt.Errorf("glob rasterized pages: %w", err)
	}
	sort.Strings(pages)
	if len(pages) == 0 {
		os.RemoveAll(tempDir)
		return "", nil, fmt.Errorf("pdfkit rasterizer produced no pages")
	}

	return tempDir, pages, nil
}

func runTesseractOCR(ctx context.Context, binary string, pages []string) (string, error) {
	parts := make([]string, 0, len(pages))

	for _, page := range pages {
		if err := ctx.Err(); err != nil {
			return "", err
		}

		imageBytes, err := os.ReadFile(page)
		if err != nil {
			return "", fmt.Errorf("read rasterized page: %w", err)
		}

		cmd := exec.CommandContext(
			ctx,
			binary,
			"stdin",
			"stdout",
			"-l",
			"eng",
			"--psm",
			"11",
			"-c",
			"preserve_interword_spaces=1",
		)
		cmd.Stdin = bytes.NewReader(imageBytes)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return "", ctxErr
			}
			if stderr.Len() > 0 {
				return "", fmt.Errorf("run tesseract: %s", strings.TrimSpace(stderr.String()))
			}
			return "", fmt.Errorf("run tesseract: %w", err)
		}

		text := strings.TrimSpace(stdout.String())
		if text != "" {
			parts = append(parts, text)
		}
	}

	return strings.Join(parts, "\n"), nil
}

type rasterizerUnavailableError struct {
	Err error
}

func (e *rasterizerUnavailableError) Error() string {
	if e == nil || e.Err == nil {
		return "rasterizer unavailable"
	}
	return e.Err.Error()
}

func (e *rasterizerUnavailableError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func isUnavailableRasterizerError(err error) bool {
	var unavailable *rasterizerUnavailableError
	return errors.As(err, &unavailable)
}

func unwrapRasterizerError(err error) error {
	var unavailable *rasterizerUnavailableError
	if errors.As(err, &unavailable) && unavailable.Err != nil {
		return unavailable.Err
	}
	return err
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
let outputDirURL = URL(fileURLWithPath: args[2], isDirectory: true)
guard let dpi = Double(args[3]) else {
    fputs("invalid dpi\n", stderr)
    exit(2)
}

guard let document = PDFDocument(url: inputURL) else {
    fputs("open pdf failed\n", stderr)
    exit(1)
}

let scale = CGFloat(dpi / 72.0)
let fileManager = FileManager.default

do {
    try fileManager.createDirectory(at: outputDirURL, withIntermediateDirectories: true)

    for index in 0..<document.pageCount {
        guard let page = document.page(at: index) else {
            continue
        }

        let bounds = page.bounds(for: .mediaBox)
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
            fputs("bitmap alloc failed\n", stderr)
            exit(1)
        }

        bitmap.size = NSSize(width: bounds.width, height: bounds.height)

        NSGraphicsContext.saveGraphicsState()
        guard let context = NSGraphicsContext(bitmapImageRep: bitmap) else {
            fputs("graphics context failed\n", stderr)
            exit(1)
        }
        NSGraphicsContext.current = context

        let cg = context.cgContext
        cg.setFillColor(NSColor.white.cgColor)
        cg.fill(CGRect(x: 0, y: 0, width: CGFloat(width), height: CGFloat(height)))
        cg.interpolationQuality = .none
        cg.scaleBy(x: scale, y: scale)
        page.draw(with: .mediaBox, to: cg)
        context.flushGraphics()
        NSGraphicsContext.restoreGraphicsState()

        guard let data = bitmap.representation(using: .png, properties: [:]) else {
            fputs("png encode failed\n", stderr)
            exit(1)
        }

        let name = String(format: "page-%04d.png", index + 1)
        try data.write(to: outputDirURL.appendingPathComponent(name))
    }
} catch {
    fputs("\(error)\n", stderr)
    exit(1)
}
`
