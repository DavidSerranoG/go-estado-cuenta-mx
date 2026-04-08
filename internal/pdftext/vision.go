package pdftext

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Vision uses macOS PDFKit + Vision OCR through the Swift runtime.
type Vision struct {
	lookPath func(string) (string, error)
	run      func(context.Context, string, []byte) (string, error)
	goos     string
}

// NewVision returns the optional macOS-native OCR extractor.
func NewVision() Vision {
	return Vision{}
}

// Name identifies the extractor in chain reports.
func (Vision) Name() string {
	return "vision"
}

// ExtractText converts a PDF into plain text using PDFKit rendering and Vision OCR.
func (v Vision) ExtractText(ctx context.Context, pdfBytes []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", newExtractorError("vision", AttemptCodeFailed, err)
	}
	if v.platform() != "darwin" {
		return "", newExtractorError("vision", AttemptCodeUnavailable, fmt.Errorf("vision OCR is only available on macOS"))
	}

	lookPath := v.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	swift, err := lookPath("swift")
	if err != nil {
		return "", newExtractorError("vision", AttemptCodeUnavailable, fmt.Errorf("find swift binary: %w", err))
	}

	run := v.run
	if run == nil {
		run = runVisionOCR
	}

	text, err := run(ctx, swift, pdfBytes)
	if err != nil {
		return "", newExtractorError("vision", AttemptCodeFailed, err)
	}

	trimmed := strings.TrimSpace(strings.ReplaceAll(text, "\x00", ""))
	if trimmed == "" {
		return "", nil
	}

	return trimmed, nil
}

func (v Vision) platform() string {
	if v.goos != "" {
		return v.goos
	}
	return runtime.GOOS
}

func runVisionOCR(ctx context.Context, binary string, pdfBytes []byte) (string, error) {
	tempDir, err := os.MkdirTemp("", "edocuenta-vision-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	inputPath := filepath.Join(tempDir, "input.pdf")
	if err := os.WriteFile(inputPath, pdfBytes, 0o600); err != nil {
		return "", fmt.Errorf("write temp pdf: %w", err)
	}

	scriptPath := filepath.Join(tempDir, "vision.swift")
	if err := os.WriteFile(scriptPath, []byte(swiftVisionOCRScript), 0o600); err != nil {
		return "", fmt.Errorf("write vision script: %w", err)
	}

	cmd := exec.CommandContext(ctx, binary, scriptPath, inputPath, "300")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%s", strings.TrimSpace(stderr.String()))
		}
		return "", err
	}

	return stdout.String(), nil
}

const swiftVisionOCRScript = `
import AppKit
import Foundation
import PDFKit
import Vision

func render(page: PDFPage, dpi: Double) -> CGImage? {
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
    cg.interpolationQuality = .none
    cg.scaleBy(x: scale, y: scale)
    page.draw(with: .mediaBox, to: cg)
    context.flushGraphics()
    NSGraphicsContext.restoreGraphicsState()

    return bitmap.cgImage
}

let args = CommandLine.arguments
guard args.count == 3 else {
    fputs("usage: vision.swift input.pdf dpi\n", stderr)
    exit(2)
}

let inputURL = URL(fileURLWithPath: args[1])
guard let dpi = Double(args[2]) else {
    fputs("invalid dpi\n", stderr)
    exit(2)
}

guard let document = PDFDocument(url: inputURL) else {
    fputs("open pdf failed\n", stderr)
    exit(1)
}

let request = VNRecognizeTextRequest()
request.recognitionLevel = .accurate
request.recognitionLanguages = ["es-MX", "en-US"]
request.usesLanguageCorrection = false

func sortedLines(_ observations: [VNRecognizedTextObservation]) -> [String] {
    let sorted = observations.sorted {
        let ay = $0.boundingBox.minY
        let by = $1.boundingBox.minY
        if abs(ay - by) > 0.01 {
            return ay > by
        }
        return $0.boundingBox.minX < $1.boundingBox.minX
    }

    return sorted.compactMap { observation in
        observation.topCandidates(1).first?.string.trimmingCharacters(in: .whitespacesAndNewlines)
    }.filter { !$0.isEmpty }
}

for index in 0..<document.pageCount {
    guard let page = document.page(at: index), let image = render(page: page, dpi: dpi) else {
        continue
    }

    let handler = VNImageRequestHandler(cgImage: image, options: [:])
    do {
        try handler.perform([request])
    } catch {
        fputs("\(error)\n", stderr)
        exit(1)
    }

    let observations = request.results ?? []
    for line in sortedLines(observations) {
        print(line)
    }
}
`
