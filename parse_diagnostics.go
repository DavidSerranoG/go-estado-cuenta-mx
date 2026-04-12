package edocuenta

import (
	"fmt"
	"strings"
	"time"
)

const txPeriodGrace = 7 * 24 * time.Hour

// LayoutParser optionally exposes the normalized layout identifier a parser matched.
type LayoutParser interface {
	Layout(text string) string
}

type validationSummary struct {
	score      int
	confidence ParseConfidence
	issues     []string
}

func buildParseDiagnostics(parser Parser, text string, result ParseResult, detectionScore int) ParseDiagnostics {
	validation := validateParseResult(result)
	return ParseDiagnostics{
		SelectedParser: parserDisplayName(parser),
		Layout:         parserLayout(parser, text, result),
		DetectionScore: detectionScore,
		Confidence:     validation.confidence,
		Issues:         validation.issues,
	}
}

func validateParseResult(result ParseResult) validationSummary {
	issues := make([]string, 0)
	score := 100
	statement := result.Statement

	if statement.Bank == "" {
		issues = append(issues, "missing bank identifier")
		score -= 40
	}
	if strings.TrimSpace(statement.AccountNumber) == "" {
		issues = append(issues, "missing account number")
		score -= 30
	}
	if statement.Currency == "" {
		issues = append(issues, "missing currency")
		score -= 10
	}
	switch {
	case statement.PeriodStart.IsZero() || statement.PeriodEnd.IsZero():
		issues = append(issues, "missing statement period")
		score -= 30
	case statement.PeriodEnd.Before(statement.PeriodStart):
		issues = append(issues, "invalid statement period")
		score -= 35
	}

	txIssues, txPenalty := validateTransactions(statement.Transactions, statement.PeriodStart, statement.PeriodEnd)
	issues = append(issues, txIssues...)
	score -= txPenalty

	if warningPenalty := len(result.Warnings) * 4; warningPenalty > 0 {
		if warningPenalty > 16 {
			warningPenalty = 16
		}
		score -= warningPenalty
	}

	if score < 0 {
		score = 0
	}

	confidence := ParseConfidenceLow
	switch {
	case score >= 85:
		confidence = ParseConfidenceHigh
	case score >= 60:
		confidence = ParseConfidenceMedium
	}

	return validationSummary{
		score:      score,
		confidence: confidence,
		issues:     issues,
	}
}

func validateTransactions(transactions []Transaction, periodStart, periodEnd time.Time) ([]string, int) {
	if len(transactions) == 0 {
		return []string{"no transactions found"}, 50
	}

	missingDates := 0
	outsidePeriod := 0
	missingDescriptions := 0
	invalidAmounts := 0
	invalidDirections := 0

	for _, tx := range transactions {
		if tx.PostedAt.IsZero() {
			missingDates++
		} else if !periodStart.IsZero() && !periodEnd.IsZero() {
			if tx.PostedAt.Before(periodStart.Add(-txPeriodGrace)) || tx.PostedAt.After(periodEnd.Add(txPeriodGrace)) {
				outsidePeriod++
			}
		}
		if strings.TrimSpace(tx.Description) == "" {
			missingDescriptions++
		}
		if tx.AmountCents <= 0 {
			invalidAmounts++
		}
		if tx.Direction != TransactionDirectionDebit && tx.Direction != TransactionDirectionCredit {
			invalidDirections++
		}
	}

	issues := make([]string, 0, 5)
	penalty := 0
	if missingDates > 0 {
		issues = append(issues, fmt.Sprintf("%d transactions missing posting date", missingDates))
		penalty += minPenalty(25, missingDates*10)
	}
	if outsidePeriod > 0 {
		issues = append(issues, fmt.Sprintf("%d transactions outside statement period", outsidePeriod))
		penalty += minPenalty(20, outsidePeriod*5)
	}
	if missingDescriptions > 0 {
		issues = append(issues, fmt.Sprintf("%d transactions missing description", missingDescriptions))
		penalty += minPenalty(15, missingDescriptions*5)
	}
	if invalidAmounts > 0 {
		issues = append(issues, fmt.Sprintf("%d transactions with invalid amount", invalidAmounts))
		penalty += minPenalty(20, invalidAmounts*10)
	}
	if invalidDirections > 0 {
		issues = append(issues, fmt.Sprintf("%d transactions with invalid direction", invalidDirections))
		penalty += minPenalty(20, invalidDirections*10)
	}

	return issues, penalty
}

func parserDisplayName(parser Parser) string {
	if parser == nil {
		return ""
	}
	if named, ok := parser.(interface{ Name() string }); ok {
		return named.Name()
	}
	return parser.Bank()
}

func parserLayout(parser Parser, text string, result ParseResult) string {
	if parser != nil {
		if layoutParser, ok := parser.(LayoutParser); ok {
			if layout := normalizeLayout(layoutParser.Layout(text)); layout != "" {
				return layout
			}
		}
	}
	return inferLayout(result)
}

func inferLayout(result ParseResult) string {
	text := strings.ToUpper(result.ExtractedText)
	switch {
	case result.Statement.AccountClass == AccountClassLiability:
		return "card"
	case strings.Contains(text, "CUENTA FLEXIBLE"):
		return "flexible"
	case strings.Contains(text, "TARJETA") || strings.Contains(text, "TU PAGO REQUERIDO ESTE PERIODO"):
		return "card"
	case result.Statement.AccountClass == AccountClassAsset:
		return "account"
	default:
		return ""
	}
}

func normalizeLayout(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func minPenalty(limit, value int) int {
	if value > limit {
		return limit
	}
	return value
}
