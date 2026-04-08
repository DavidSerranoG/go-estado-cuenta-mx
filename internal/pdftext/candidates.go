package pdftext

import (
	"context"
	"regexp"
	"strings"

	"github.com/DavidSerranoG/go-estado-cuenta-mx/internal/normalize"
)

var (
	qualityDateHintPattern = regexp.MustCompile(`(?i)([0-9]{2}/[0-9]{2}/[0-9]{4}|[0-9]{1,2}-[a-z0-9]{3,4}-[0-9]{4}|[0-9]{2}\s*/\s*(?:ENE|FEB|MAR|ABR|MAY|JUN|JUL|AGO|SEP|OCT|NOV|DIC))`)
	qualityAmountPattern   = regexp.MustCompile(`[$]?\s*[0-9][0-9, .]{0,18}(?:\.[0-9]{2}|,[0-9]{2}| [0-9]{2})`)
)

// CandidateExtractor exposes all usable extraction candidates, not just the
// first one.
type CandidateExtractor interface {
	ExtractCandidates(ctx context.Context, pdfBytes []byte) (CandidateRun, error)
}

// CandidateRun contains all extraction attempts plus every non-empty candidate
// text produced during the run.
type CandidateRun struct {
	Candidates []Candidate
	Attempts   []Attempt
}

// Candidate is one extracted text variant produced by a concrete extractor.
type Candidate struct {
	Extractor      string
	RawText        string
	NormalizedText string
	Quality        QualitySignals
}

// QualitySignals summarize heuristics that help compare multiple extracted
// text candidates.
type QualitySignals struct {
	LineCount     int
	AmountHints   int
	DateHints     int
	WordChars     int
	DigitChars    int
	RepeatedLines int
}

// Score returns a stable quality score for extractor candidate comparison.
func (q QualitySignals) Score() int {
	score := 0
	score += q.LineCount * 2
	score += q.AmountHints * 8
	score += q.DateHints * 10
	score += min(q.WordChars, 2000) / 20
	score += min(q.DigitChars, 2000) / 25
	score -= q.RepeatedLines * 3
	return score
}

// NewCandidate builds a candidate from raw text using the shared OCR-aware
// normalizer and quality analyzer.
func NewCandidate(name, rawText string) Candidate {
	normalized := normalize.NormalizeExtractedText(rawText)
	return Candidate{
		Extractor:      name,
		RawText:        rawText,
		NormalizedText: normalized,
		Quality:        AnalyzeQuality(normalized),
	}
}

// AnalyzeQuality summarizes parse-relevant signals for an extracted text.
func AnalyzeQuality(text string) QualitySignals {
	lines := strings.Split(text, "\n")
	seen := make(map[string]int, len(lines))
	signals := QualitySignals{}

	for _, rawLine := range lines {
		line := normalize.CollapseWhitespace(rawLine)
		if line == "" {
			continue
		}

		signals.LineCount++
		signals.AmountHints += len(qualityAmountPattern.FindAllString(line, -1))
		signals.DateHints += len(qualityDateHintPattern.FindAllString(line, -1))

		for _, r := range line {
			switch {
			case r >= '0' && r <= '9':
				signals.DigitChars++
			case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z':
				signals.WordChars++
			}
		}

		key := strings.ToUpper(line)
		seen[key]++
		if seen[key] > 1 {
			signals.RepeatedLines++
		}
	}

	return signals
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
