package edocuenta

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/DavidSerranoG/go-estado-cuenta-mx/internal/normalize"
	"github.com/DavidSerranoG/go-estado-cuenta-mx/internal/pdftext"
)

type extractorCandidate struct {
	Extractor  string
	RawText    string
	Text       string
	Quality    pdftext.QualitySignals
	UsedRescue bool
	Order      int
}

type candidateOutcome struct {
	candidate      extractorCandidate
	result         ParseResult
	detectionScore int
	err            error
}

func (p *Processor) parsePDFResult(ctx context.Context, pdfBytes []byte, bank string) (ParseResult, error) {
	if len(pdfBytes) == 0 {
		return ParseResult{}, ErrEmptyPDF
	}

	fixedParser, err := p.fixedParser(bank)
	if err != nil {
		return ParseResult{}, err
	}

	candidates, attempts, err := collectCandidates(ctx, p.extractor, pdfBytes, false, 0)
	if err != nil {
		return ParseResult{}, err
	}

	nextOrder := len(candidates)
	if p.rescueExtractor != nil {
		rescueCandidates, rescueAttempts, err := collectCandidates(ctx, p.rescueExtractor, pdfBytes, true, nextOrder)
		if err != nil {
			return ParseResult{}, err
		}
		candidates = append(candidates, rescueCandidates...)
		attempts = append(attempts, rescueAttempts...)
	}

	if len(candidates) == 0 {
		if len(attempts) == 0 {
			return ParseResult{}, wrapTextExtractionError(&pdftext.ExtractionError{})
		}
		return ParseResult{}, newTextExtractionError(attempts)
	}

	outcome, ok := p.bestCandidateOutcome(candidates, fixedParser)
	if !ok {
		return ParseResult{}, outcome.err
	}

	result := outcome.result
	result.Extraction = ExtractionDiagnostics{
		SelectedExtractor: outcome.candidate.Extractor,
		UsedRescue:        outcome.candidate.UsedRescue,
		Attempts:          publicAttempts(attempts),
	}
	result.ExtractedText = outcome.candidate.RawText
	return result, nil
}

func (p *Processor) parseTextCandidate(text string, bank string) (ParseResult, error) {
	fixedParser, err := p.fixedParser(bank)
	if err != nil {
		return ParseResult{}, err
	}

	normalized := normalize.NormalizeExtractedText(text)
	candidate := extractorCandidate{
		Extractor: "input",
		RawText:   text,
		Text:      normalized,
		Quality:   pdftext.AnalyzeQuality(normalized),
		Order:     0,
	}

	outcome := p.evaluateCandidate(candidate, fixedParser)
	if outcome.err != nil {
		return ParseResult{}, outcome.err
	}

	result := outcome.result
	result.Extraction = ExtractionDiagnostics{
		SelectedExtractor: "input",
	}
	result.ExtractedText = text
	return result, nil
}

func collectCandidates(ctx context.Context, extractor TextExtractor, pdfBytes []byte, usedRescue bool, startOrder int) ([]extractorCandidate, []pdftext.Attempt, error) {
	if extractor == nil {
		return nil, nil, nil
	}

	if candidateExtractor, ok := extractor.(interface {
		ExtractCandidates(context.Context, []byte) (pdftext.CandidateRun, error)
	}); ok {
		run, err := candidateExtractor.ExtractCandidates(ctx, pdfBytes)
		if err != nil {
			return nil, nil, err
		}
		candidates := make([]extractorCandidate, 0, len(run.Candidates))
		for i, candidate := range run.Candidates {
			candidates = append(candidates, extractorCandidate{
				Extractor:  candidate.Extractor,
				RawText:    candidate.RawText,
				Text:       candidate.NormalizedText,
				Quality:    candidate.Quality,
				UsedRescue: usedRescue,
				Order:      startOrder + i,
			})
		}
		return candidates, run.Attempts, nil
	}

	name := extractorDisplayName(extractor)
	text, err := extractor.ExtractText(ctx, pdfBytes)
	switch {
	case err == nil && strings.TrimSpace(text) != "":
		candidate := pdftext.NewCandidate(name, text)
		return []extractorCandidate{{
			Extractor:  candidate.Extractor,
			RawText:    candidate.RawText,
			Text:       candidate.NormalizedText,
			Quality:    candidate.Quality,
			UsedRescue: usedRescue,
			Order:      startOrder,
		}}, []pdftext.Attempt{{Extractor: name, Code: pdftext.AttemptCodeSucceeded}}, nil
	case err == nil:
		return nil, []pdftext.Attempt{{
			Extractor: name,
			Code:      pdftext.AttemptCodeNoText,
			Err:       errors.New("extractor returned empty text"),
		}}, nil
	default:
		return nil, []pdftext.Attempt{candidateAttemptFromError(name, err)}, nil
	}
}

func (p *Processor) bestCandidateOutcome(candidates []extractorCandidate, fixedParser Parser) (candidateOutcome, bool) {
	var best candidateOutcome
	found := false

	for _, candidate := range candidates {
		outcome := p.evaluateCandidate(candidate, fixedParser)
		if !found || compareCandidateOutcomes(outcome, best) > 0 {
			best = outcome
			found = true
		}
	}

	if !found {
		return candidateOutcome{}, false
	}

	return best, best.err == nil
}

func (p *Processor) evaluateCandidate(candidate extractorCandidate, fixedParser Parser) candidateOutcome {
	outcome := candidateOutcome{candidate: candidate}

	parser := fixedParser
	if parser == nil {
		var err error
		parser, outcome.detectionScore, err = p.detectWithScore(candidate.Text)
		if err != nil {
			outcome.err = err
			return outcome
		}
	} else {
		outcome.detectionScore = parserDetectionScore(parser, candidate.Text)
	}

	result, err := p.parseWithParser(parser, candidate.Text)
	if err != nil {
		outcome.err = err
		return outcome
	}

	outcome.result = result
	return outcome
}

func compareCandidateOutcomes(left, right candidateOutcome) int {
	if delta := compareBool(left.err == nil, right.err == nil); delta != 0 {
		return delta
	}
	if delta := compareInt(len(left.result.Statement.Transactions), len(right.result.Statement.Transactions)); delta != 0 {
		return delta
	}
	if delta := compareInt(left.detectionScore, right.detectionScore); delta != 0 {
		return delta
	}
	if delta := compareInt(left.candidate.Quality.Score(), right.candidate.Quality.Score()); delta != 0 {
		return delta
	}
	if left.candidate.Order < right.candidate.Order {
		return 1
	}
	if left.candidate.Order > right.candidate.Order {
		return -1
	}
	return 0
}

func compareBool(left, right bool) int {
	switch {
	case left && !right:
		return 1
	case !left && right:
		return -1
	default:
		return 0
	}
}

func compareInt(left, right int) int {
	switch {
	case left > right:
		return 1
	case left < right:
		return -1
	default:
		return 0
	}
}

func candidateAttemptFromError(name string, err error) pdftext.Attempt {
	var extractorErr *pdftext.ExtractorError
	if errors.As(err, &extractorErr) {
		return pdftext.Attempt{
			Extractor: extractorErr.Extractor,
			Code:      extractorErr.Code,
			Err:       extractorErr.Err,
		}
	}

	var extractionErr *pdftext.ExtractionError
	if errors.As(err, &extractionErr) && len(extractionErr.Attempts) > 0 {
		return extractionErr.Attempts[0]
	}

	return pdftext.Attempt{
		Extractor: name,
		Code:      pdftext.AttemptCodeFailed,
		Err:       err,
	}
}

func extractorDisplayName(extractor TextExtractor) string {
	if named, ok := extractor.(interface{ Name() string }); ok {
		return named.Name()
	}
	return fmt.Sprintf("%T", extractor)
}

func (p *Processor) fixedParser(bank string) (Parser, error) {
	if strings.TrimSpace(bank) == "" {
		return nil, nil
	}
	return p.byBank(bank)
}

func (p *Processor) detectWithScore(text string) (Parser, int, error) {
	if p.parsers.Len() == 0 {
		return nil, 0, ErrNoParsersConfigured
	}

	parser, score, ok := p.parsers.FindByTextWithScore(text)
	if !ok {
		return nil, 0, ErrUnsupportedFormat
	}

	return parser, score, nil
}

func parserDetectionScore(parser Parser, text string) int {
	if scored, ok := parser.(ScoredParser); ok {
		return scored.DetectionScore(text)
	}
	if parser.CanParse(text) {
		return 1
	}
	return 0
}
