package bbva

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ledgermx/mxstatementpdf"
	"github.com/ledgermx/mxstatementpdf/internal/normalize"
)

var (
	accountPattern = regexp.MustCompile(`(?i)cuenta:\s*([0-9]+)`)
	periodPattern  = regexp.MustCompile(`(?i)periodo:\s*([0-9]{2}/[0-9]{2}/[0-9]{4})\s*(?:-|al)\s*([0-9]{2}/[0-9]{2}/[0-9]{4})`)
	txPattern      = regexp.MustCompile(`(?i)^([0-9]{2}/[0-9]{2}/[0-9]{4})\s+(.+?)\s+(ABONO|CARGO)\s+([0-9,]+\.\d{2})\s+([0-9,]+\.\d{2})$`)
)

// Parser parses BBVA bank statements.
type Parser struct{}

// New returns a new BBVA parser.
func New() Parser {
	return Parser{}
}

// Bank returns the canonical bank identifier.
func (Parser) Bank() string {
	return "bbva"
}

// CanParse checks whether the extracted text looks like BBVA.
func (Parser) CanParse(text string) bool {
	return strings.Contains(strings.ToUpper(text), "BBVA")
}

// Parse parses extracted plain text into a normalized statement.
func (Parser) Parse(text string) (statementpdf.Statement, error) {
	account, err := parseAccount(text)
	if err != nil {
		return statementpdf.Statement{}, err
	}

	periodStart, periodEnd, err := parsePeriod(text)
	if err != nil {
		return statementpdf.Statement{}, err
	}

	var (
		transactions []statementpdf.Transaction
		warnings     []string
	)

	for _, rawLine := range strings.Split(text, "\n") {
		line := normalize.CollapseWhitespace(rawLine)
		if line == "" {
			continue
		}

		match := txPattern.FindStringSubmatch(line)
		if len(match) != 6 {
			if looksLikeTransaction(line) {
				warnings = append(warnings, "line ignored: "+line)
			}
			continue
		}

		postedAt, err := normalize.ParseDateDDMMYYYY(match[1])
		if err != nil {
			warnings = append(warnings, "invalid date: "+line)
			continue
		}

		amountCents, err := normalize.ParseMoneyToCents(match[4])
		if err != nil {
			warnings = append(warnings, "invalid amount: "+line)
			continue
		}

		balanceCents, err := normalize.ParseMoneyToCents(match[5])
		if err != nil {
			warnings = append(warnings, "invalid balance: "+line)
			continue
		}

		balanceCopy := balanceCents
		transactions = append(transactions, statementpdf.Transaction{
			PostedAt:     postedAt,
			Description:  match[2],
			Type:         strings.ToLower(match[3]),
			AmountCents:  amountCents,
			BalanceCents: &balanceCopy,
			RawLine:      line,
		})
	}

	if len(transactions) == 0 {
		return statementpdf.Statement{}, fmt.Errorf("bbva: no transactions found")
	}

	return statementpdf.Statement{
		Bank:          "bbva",
		AccountNumber: account,
		Currency:      "MXN",
		PeriodStart:   periodStart,
		PeriodEnd:     periodEnd,
		Transactions:  transactions,
		Warnings:      warnings,
	}, nil
}

func parseAccount(text string) (string, error) {
	match := accountPattern.FindStringSubmatch(text)
	if len(match) != 2 {
		return "", fmt.Errorf("bbva: account number not found")
	}

	return match[1], nil
}

func parsePeriod(text string) (time.Time, time.Time, error) {
	match := periodPattern.FindStringSubmatch(text)
	if len(match) != 3 {
		return time.Time{}, time.Time{}, fmt.Errorf("bbva: period not found")
	}

	start, err := normalize.ParseDateDDMMYYYY(match[1])
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	end, err := normalize.ParseDateDDMMYYYY(match[2])
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	return start, end, nil
}

func looksLikeTransaction(line string) bool {
	if len(line) < 1 {
		return false
	}

	return line[0] >= '0' && line[0] <= '9'
}
