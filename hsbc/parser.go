package hsbc

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ledgermx/mxstatementpdf"
	"github.com/ledgermx/mxstatementpdf/internal/normalize"
)

var (
	cardAccountPattern     = regexp.MustCompile(`(?i)n[úu]mero de cuenta:\s*([0-9 ]{16,})`)
	cardPeriodPattern      = regexp.MustCompile(`(?i)([0-9]{2}-[a-z]{3}-[0-9]{4})\s+al\s+([0-9]{2}-[a-z]{3}-[0-9]{4})`)
	cardFullTxPattern      = regexp.MustCompile(`^([0-9]{2}-[A-Za-z]{3}-[0-9]{4})\s*([0-9]{2}-[A-Za-z]{3}-[0-9]{4})\s*(.+?)([+-])\s*\$\s*([0-9,]+\.\d{2})$`)
	cardOpenTxPattern      = regexp.MustCompile(`^([0-9]{2}-[A-Za-z]{3}-[0-9]{4})\s*([0-9]{2}-[A-Za-z]{3}-[0-9]{4})\s*(.+)$`)
	cardAmountPattern      = regexp.MustCompile(`(.+?)([+-])\s*\$\s*([0-9,]+\.\d{2})$`)
	flexibleAccountPattern = regexp.MustCompile(`(?i)detalle movimientos cuenta flexible no\.\s*([0-9]{10})`)
	flexiblePeriodPattern  = regexp.MustCompile(`(?i)per[íi]odo del\s*([0-9]{8})\s*al\s*([0-9]{8})`)
	flexibleInitialPattern = regexp.MustCompile(`(?is)saldo inicial del\s*periodo\s*\$\s*([0-9,]+\.\d{2})`)
	flexibleAmountPattern  = regexp.MustCompile(`^\$\s*([0-9,]+\.\d{2})\s+\$\s*([0-9,]+\.\d{2})$`)
	flexibleHeaderPattern  = regexp.MustCompile(`^\s*(.+?)\s{2,}([A-Z0-9]+)\s*$`)
	flexibleRefPattern     = regexp.MustCompile(`^(.*)\s+([A-Z0-9]{8,})$`)
)

// Parser parses HSBC bank statements.
type Parser struct{}

// New returns a new HSBC parser.
func New() Parser {
	return Parser{}
}

// Bank returns the canonical bank identifier.
func (Parser) Bank() string {
	return "hsbc"
}

// CanParse checks whether the extracted text looks like HSBC.
func (Parser) CanParse(text string) bool {
	upper := strings.ToUpper(text)
	return strings.Contains(upper, "HSBC") && strings.Contains(upper, "NÚMERO DE CUENTA")
}

// Parse parses extracted plain text into a normalized statement.
func (Parser) Parse(text string) (statementpdf.Statement, error) {
	if strings.Contains(strings.ToUpper(text), "CUENTA FLEXIBLE") {
		return parseFlexible(text)
	}

	return parseCard(text)
}

func parseCard(text string) (statementpdf.Statement, error) {
	account, err := parseCardAccount(text)
	if err != nil {
		return statementpdf.Statement{}, err
	}

	periodStart, periodEnd, err := parseCardPeriod(text)
	if err != nil {
		return statementpdf.Statement{}, err
	}

	var (
		transactions []statementpdf.Transaction
		warnings     []string
		pending      *pendingTransaction
	)

	for _, rawLine := range strings.Split(text, "\n") {
		line := normalize.CollapseWhitespace(rawLine)
		if line == "" {
			continue
		}

		if pending != nil {
			transaction, consumed, err := completePending(*pending, line)
			if err == nil && consumed {
				transactions = append(transactions, transaction)
				pending = nil
				continue
			}
			if looksLikeContinuation(line) {
				pending.description = normalize.CollapseWhitespace(pending.description + " " + line)
				pending.rawLine = pending.rawLine + " | " + line
				continue
			}
			warnings = append(warnings, "line ignored: "+pending.rawLine)
			pending = nil
		}

		if transaction, ok, err := parseFullTransaction(line); ok {
			if err != nil {
				warnings = append(warnings, err.Error()+": "+line)
				continue
			}
			transactions = append(transactions, transaction)
			continue
		}

		if tx, ok, err := parseOpenTransaction(line); ok {
			if err != nil {
				warnings = append(warnings, err.Error()+": "+line)
				continue
			}
			pending = &tx
			continue
		}
	}

	if pending != nil {
		warnings = append(warnings, "line ignored: "+pending.rawLine)
	}

	if len(transactions) == 0 {
		return statementpdf.Statement{}, fmt.Errorf("hsbc: no transactions found")
	}

	return statementpdf.Statement{
		Bank:          "hsbc",
		AccountNumber: account,
		Currency:      "MXN",
		PeriodStart:   periodStart,
		PeriodEnd:     periodEnd,
		Transactions:  transactions,
		Warnings:      warnings,
	}, nil
}

func parseAccount(text string) (string, error) {
	match := cardAccountPattern.FindStringSubmatch(text)
	if len(match) != 2 {
		return "", fmt.Errorf("hsbc: account number not found")
	}

	return strings.ReplaceAll(match[1], " ", ""), nil
}

func parseCardAccount(text string) (string, error) {
	return parseAccount(text)
}

func parseCardPeriod(text string) (periodStart, periodEnd time.Time, err error) {
	match := cardPeriodPattern.FindStringSubmatch(text)
	if len(match) != 3 {
		return time.Time{}, time.Time{}, fmt.Errorf("hsbc: period not found")
	}

	periodStart, err = parseSpanishDate(match[1])
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	periodEnd, err = parseSpanishDate(match[2])
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	return periodStart, periodEnd, nil
}

type pendingTransaction struct {
	postedAt    time.Time
	description string
	rawLine     string
}

func parseFullTransaction(line string) (statementpdf.Transaction, bool, error) {
	match := cardFullTxPattern.FindStringSubmatch(line)
	if len(match) != 6 {
		return statementpdf.Transaction{}, false, nil
	}

	postedAt, err := parseSpanishDate(match[2])
	if err != nil {
		return statementpdf.Transaction{}, true, fmt.Errorf("hsbc: invalid posting date")
	}

	amountCents, err := normalize.ParseMoneyToCents(match[5])
	if err != nil {
		return statementpdf.Transaction{}, true, fmt.Errorf("hsbc: invalid amount")
	}

	return statementpdf.Transaction{
		PostedAt:    postedAt,
		Description: normalize.CollapseWhitespace(match[3]),
		Type:        movementType(match[4]),
		AmountCents: amountCents,
		RawLine:     line,
	}, true, nil
}

func parseOpenTransaction(line string) (pendingTransaction, bool, error) {
	match := cardOpenTxPattern.FindStringSubmatch(line)
	if len(match) != 4 {
		return pendingTransaction{}, false, nil
	}
	if cardAmountPattern.MatchString(line) {
		return pendingTransaction{}, false, nil
	}

	postedAt, err := parseSpanishDate(match[2])
	if err != nil {
		return pendingTransaction{}, true, fmt.Errorf("hsbc: invalid posting date")
	}

	return pendingTransaction{
		postedAt:    postedAt,
		description: normalize.CollapseWhitespace(match[3]),
		rawLine:     line,
	}, true, nil
}

func completePending(pending pendingTransaction, line string) (statementpdf.Transaction, bool, error) {
	match := cardAmountPattern.FindStringSubmatch(line)
	if len(match) != 4 {
		return statementpdf.Transaction{}, false, nil
	}

	amountCents, err := normalize.ParseMoneyToCents(match[3])
	if err != nil {
		return statementpdf.Transaction{}, true, fmt.Errorf("hsbc: invalid amount")
	}

	description := pending.description
	continuation := normalize.CollapseWhitespace(match[1])
	if continuation != "" {
		description = normalize.CollapseWhitespace(description + " " + continuation)
	}

	return statementpdf.Transaction{
		PostedAt:    pending.postedAt,
		Description: description,
		Type:        movementType(match[2]),
		AmountCents: amountCents,
		RawLine:     pending.rawLine + " | " + line,
	}, true, nil
}

func parseSpanishDate(value string) (time.Time, error) {
	parts := strings.Split(strings.TrimSpace(value), "-")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("hsbc: invalid date %q", value)
	}

	day, err := parseDay(parts[0])
	if err != nil {
		return time.Time{}, err
	}
	month, err := parseMonth(parts[1])
	if err != nil {
		return time.Time{}, err
	}
	year, err := parseYear(parts[2])
	if err != nil {
		return time.Time{}, err
	}

	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), nil
}

func parseDay(value string) (int, error) {
	if len(value) != 2 {
		return 0, fmt.Errorf("hsbc: invalid day %q", value)
	}
	return strconv.Atoi(value)
}

func parseYear(value string) (int, error) {
	if len(value) != 4 {
		return 0, fmt.Errorf("hsbc: invalid year %q", value)
	}
	return strconv.Atoi(value)
}

func parseMonth(value string) (time.Month, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ene":
		return time.January, nil
	case "feb":
		return time.February, nil
	case "mar":
		return time.March, nil
	case "abr":
		return time.April, nil
	case "may":
		return time.May, nil
	case "jun":
		return time.June, nil
	case "jul":
		return time.July, nil
	case "ago":
		return time.August, nil
	case "sep":
		return time.September, nil
	case "oct":
		return time.October, nil
	case "nov":
		return time.November, nil
	case "dic":
		return time.December, nil
	default:
		return 0, fmt.Errorf("hsbc: invalid month %q", value)
	}
}

func movementType(sign string) string {
	if sign == "-" {
		return "abono"
	}
	return "cargo"
}

func looksLikeContinuation(line string) bool {
	if line == "" {
		return false
	}

	return !cardOpenTxPattern.MatchString(line)
}

func parseFlexible(text string) (statementpdf.Statement, error) {
	account, err := parseFlexibleAccount(text)
	if err != nil {
		return statementpdf.Statement{}, err
	}

	periodStart, periodEnd, err := parseFlexiblePeriod(text)
	if err != nil {
		return statementpdf.Statement{}, err
	}

	initialBalance, err := parseFlexibleInitialBalance(text)
	if err != nil {
		return statementpdf.Statement{}, err
	}

	transactions, warnings, err := parseFlexibleTransactions(text, periodStart, periodEnd, initialBalance)
	if err != nil {
		return statementpdf.Statement{}, err
	}

	return statementpdf.Statement{
		Bank:          "hsbc",
		AccountNumber: account,
		Currency:      "MXN",
		PeriodStart:   periodStart,
		PeriodEnd:     periodEnd,
		Transactions:  transactions,
		Warnings:      warnings,
	}, nil
}

func parseFlexibleAccount(text string) (string, error) {
	match := flexibleAccountPattern.FindStringSubmatch(text)
	if len(match) == 2 {
		return match[1], nil
	}

	return "", fmt.Errorf("hsbc: account number not found")
}

func parseFlexiblePeriod(text string) (time.Time, time.Time, error) {
	match := flexiblePeriodPattern.FindStringSubmatch(text)
	if len(match) != 3 {
		return time.Time{}, time.Time{}, fmt.Errorf("hsbc: period not found")
	}

	start, err := parseCompactDate(match[1])
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	end, err := parseCompactDate(match[2])
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	return start, end, nil
}

func parseFlexibleInitialBalance(text string) (int64, error) {
	match := flexibleInitialPattern.FindStringSubmatch(text)
	if len(match) != 2 {
		return 0, fmt.Errorf("hsbc: initial balance not found")
	}

	return normalize.ParseMoneyToCents(match[1])
}

func parseFlexibleTransactions(text string, periodStart, periodEnd time.Time, initialBalance int64) ([]statementpdf.Transaction, []string, error) {
	lines := strings.Split(text, "\n")
	var (
		transactions []statementpdf.Transaction
		warnings     []string
		inSection    bool
		prevBalance  = initialBalance
	)

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		upper := strings.ToUpper(line)
		if !inSection {
			if strings.Contains(upper, "DETALLE MOVIMIENTOS CUENTA FLEXIBLE") {
				inSection = true
			}
			continue
		}

		if strings.HasPrefix(upper, "SALDO INICIAL $") || strings.HasPrefix(upper, "INFORMACIÓNSPEI") || strings.HasPrefix(upper, "INFORMACIONSPEI") {
			break
		}

		if !looksLikeFlexibleHeader(line) {
			continue
		}

		header := line
		description, reference := parseFlexibleDescriptionAndReference(header[2:])
		postedAt, err := inferFlexibleDate(strings.TrimSpace(header[:2]), periodStart, periodEnd)
		if err != nil {
			warnings = append(warnings, "invalid date: "+line)
			continue
		}

		var serial string
		var amountLine string
		for j := i + 1; j < len(lines); j++ {
			next := strings.TrimSpace(lines[j])
			if next == "" {
				continue
			}
			if flexibleAmountPattern.MatchString(next) {
				amountLine = next
				i = j
				break
			}
			if serial == "" && isFlexibleSerial(next) {
				serial = next
				continue
			}
			if looksLikeFlexibleHeader(next) {
				warnings = append(warnings, "line ignored: "+line)
				i = j - 1
				break
			}
			description = normalize.CollapseWhitespace(description + " " + next)
		}

		if amountLine == "" {
			warnings = append(warnings, "line ignored: "+line)
			continue
		}

		match := flexibleAmountPattern.FindStringSubmatch(amountLine)
		if len(match) != 3 {
			warnings = append(warnings, "invalid amount line: "+amountLine)
			continue
		}

		amountCents, err := normalize.ParseMoneyToCents(match[1])
		if err != nil {
			warnings = append(warnings, "invalid amount: "+amountLine)
			continue
		}
		balanceCents, err := normalize.ParseMoneyToCents(match[2])
		if err != nil {
			warnings = append(warnings, "invalid balance: "+amountLine)
			continue
		}

		txType := inferFlexibleMovementType(prevBalance, amountCents, balanceCents, description)
		balanceCopy := balanceCents
		txReference := reference
		if serial != "" {
			if txReference != "" {
				txReference = txReference + "/" + serial
			} else {
				txReference = serial
			}
		}

		transactions = append(transactions, statementpdf.Transaction{
			PostedAt:     postedAt,
			Description:  description,
			Reference:    txReference,
			Type:         txType,
			AmountCents:  amountCents,
			BalanceCents: &balanceCopy,
			RawLine:      strings.TrimSpace(line) + " | " + amountLine,
		})

		prevBalance = balanceCents
	}

	if len(transactions) == 0 {
		return nil, warnings, fmt.Errorf("hsbc: no transactions found")
	}

	return transactions, warnings, nil
}

func looksLikeFlexibleHeader(line string) bool {
	line = strings.TrimSpace(line)
	if len(line) < 3 {
		return false
	}

	return line[0] >= '0' && line[0] <= '9' && line[1] >= '0' && line[1] <= '9'
}

func isFlexibleSerial(line string) bool {
	if len(line) < 4 {
		return false
	}
	for _, r := range line {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func parseFlexibleDescriptionAndReference(value string) (string, string) {
	raw := strings.TrimSpace(value)
	if match := flexibleHeaderPattern.FindStringSubmatch(raw); len(match) == 3 {
		return normalize.CollapseWhitespace(match[1]), match[2]
	}
	if match := flexibleRefPattern.FindStringSubmatch(raw); len(match) == 3 {
		return normalize.CollapseWhitespace(match[1]), match[2]
	}
	return normalize.CollapseWhitespace(raw), ""
}

func inferFlexibleDate(day string, periodStart, periodEnd time.Time) (time.Time, error) {
	dayInt, err := strconv.Atoi(day)
	if err != nil {
		return time.Time{}, err
	}

	if periodStart.Month() == periodEnd.Month() && periodStart.Year() == periodEnd.Year() {
		return time.Date(periodStart.Year(), periodStart.Month(), dayInt, 0, 0, 0, 0, time.UTC), nil
	}

	if dayInt >= periodStart.Day() {
		return time.Date(periodStart.Year(), periodStart.Month(), dayInt, 0, 0, 0, 0, time.UTC), nil
	}

	return time.Date(periodEnd.Year(), periodEnd.Month(), dayInt, 0, 0, 0, 0, time.UTC), nil
}

func inferFlexibleMovementType(prevBalance, amountCents, balanceCents int64, description string) string {
	switch {
	case prevBalance+amountCents == balanceCents:
		return "abono"
	case prevBalance-amountCents == balanceCents:
		return "cargo"
	case strings.Contains(strings.ToUpper(description), "PAGO DE TARJETA"):
		return "cargo"
	default:
		return "abono"
	}
}

func parseCompactDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if len(value) != 8 {
		return time.Time{}, fmt.Errorf("hsbc: invalid compact date %q", value)
	}

	day, err := strconv.Atoi(value[:2])
	if err != nil {
		return time.Time{}, err
	}
	month, err := strconv.Atoi(value[2:4])
	if err != nil {
		return time.Time{}, err
	}
	year, err := strconv.Atoi(value[4:])
	if err != nil {
		return time.Time{}, err
	}

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
}
