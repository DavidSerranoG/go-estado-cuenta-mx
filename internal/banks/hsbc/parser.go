package hsbc

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	edocuenta "github.com/DavidSerranoG/go-estado-cuenta-mx"
	"github.com/DavidSerranoG/go-estado-cuenta-mx/internal/normalize"
)

var (
	cardAccountPattern     = regexp.MustCompile(`(?i)n[úu]mero\s*de\s*cuenta:?\s*([0-9 ]{16,})`)
	cardPeriodPattern      = regexp.MustCompile(`(?i)([0-9]{2}-[a-z0-9]{3,4}-[0-9]{4})\s*al\s*([0-9]{2}-[a-z0-9]{3,4}-[0-9]{4})`)
	cardFullTxPattern      = regexp.MustCompile(`^([0-9]{2}-[A-Za-z0-9]{3,4}-[0-9]{4})\s*([0-9]{2}-[A-Za-z0-9]{3,4}-[0-9]{4})\s*(.+?)([+-])\s*\$\s*([0-9,]+\.\d{2})$`)
	cardOpenTxPattern      = regexp.MustCompile(`^([0-9]{2}-[A-Za-z0-9]{3,4}-[0-9]{4})\s*([0-9]{2}-[A-Za-z0-9]{3,4}-[0-9]{4})\s*(.+)$`)
	cardAmountPattern      = regexp.MustCompile(`(.+?)([+-])\s*\$\s*([0-9,]+\.\d{2})$`)
	cardOCRLinePattern     = regexp.MustCompile(`^([0-9A-Za-z]{1,2}-[A-Za-z0-9]{3,4}-[0-9]{4})\s+([0-9A-Za-z]{1,2}-[A-Za-z0-9]{3,4}-[0-9]{4})\s+(.+)$`)
	cardOCRAmountPattern   = regexp.MustCompile(`\s*([+-])?\s*\$?\s*([0-9][0-9, ]*(?:\.[0-9]{2}|,[0-9]{2}| [0-9]{2}))\s*$`)
	cardOCRFXPattern       = regexp.MustCompile(`(?i)moneda extranjera:\s*([0-9]+(?:[.,][0-9]{2})?)\s*usd\s*tc:\s*([0-9]+(?:[.,][0-9]{1,4})?)`)
	flexibleAccountPattern = regexp.MustCompile(`(?i)detalle movimientos cuenta flexible no\.\s*([0-9]{10})`)
	flexiblePeriodPattern  = regexp.MustCompile(`(?i)per[íi]odo(?:\s+de(?:l)?)?\s*([0-9]{8}|[0-9]{2}\s*/\s*[0-9]{2}\s*/\s*[0-9]{4})\s*al\s*([0-9]{8}|[0-9]{2}\s*/\s*[0-9]{2}\s*/\s*[0-9]{4})`)
	flexibleInitialPattern = regexp.MustCompile(`(?is)saldo inicial del\s*periodo\s*\$\s*([0-9,]+\.\d{2})`)
	flexibleClosingPattern = regexp.MustCompile(`(?i)saldo final\s*\$\s*([0-9,]+\.\d{2})`)
	flexibleMoneyPattern   = regexp.MustCompile(`\$\s*([0-9,]+\.\d{2})`)
	flexibleSingleAmount   = regexp.MustCompile(`^\$\s*([0-9,]+\.\d{2})$`)
	flexibleAmountPattern  = regexp.MustCompile(`^\$\s*([0-9,]+\.\d{2})\s+\$\s*([0-9,]+\.\d{2})$`)
	flexibleHeaderPattern  = regexp.MustCompile(`^\s*(.+?)\s{2,}([A-Z0-9]+)\s*$`)
	flexibleRefPattern     = regexp.MustCompile(`^(.*)\s+([A-Z0-9]{8,})$`)
)

const hsbcDetectThreshold = 6

// Parser parses HSBC bank statements.
type Parser struct{}

// New returns a new HSBC parser.
func New() Parser {
	return Parser{}
}

// Bank returns the canonical bank identifier.
func (Parser) Bank() string {
	return string(edocuenta.BankHSBC)
}

// DetectionScore returns a structural confidence score for HSBC layouts.
func (Parser) DetectionScore(text string) int {
	return hsbcDetectionScore(text)
}

// Layout returns the normalized HSBC layout identifier.
func (Parser) Layout(text string) string {
	text = normalize.NormalizeExtractedText(text)
	if strings.Contains(strings.ToUpper(text), "CUENTA FLEXIBLE") {
		return "flexible"
	}
	return "card"
}

// CanParse checks whether the extracted text looks like HSBC.
func (Parser) CanParse(text string) bool {
	return hsbcDetectionScore(text) > 0
}

// Parse parses extracted plain text into a normalized statement.
func (Parser) Parse(text string) (edocuenta.Statement, error) {
	result, err := Parser{}.ParseResult(text)
	if err != nil {
		return edocuenta.Statement{}, err
	}

	return result.Statement, nil
}

// ParseResult parses extracted plain text into a normalized statement plus
// best-effort parser warnings.
func (Parser) ParseResult(text string) (edocuenta.ParseResult, error) {
	text = normalize.NormalizeExtractedText(text)

	if strings.Contains(strings.ToUpper(text), "CUENTA FLEXIBLE") {
		return parseFlexibleResult(text)
	}

	return parseCardResult(text)
}

func parseCardResult(text string) (edocuenta.ParseResult, error) {
	account, err := parseCardAccount(text)
	if err != nil {
		return edocuenta.ParseResult{}, err
	}

	periodStart, periodEnd, err := parseCardPeriod(text)
	if err != nil {
		return edocuenta.ParseResult{}, err
	}

	transactions, warnings, exactErr := parseCardTransactions(text)
	ocrTransactions, ocrWarnings, ocrErr := parseOCRCardTransactions(text, periodStart, periodEnd)

	switch {
	case ocrErr == nil && (exactErr != nil || (looksLikeOCRCardText(text) && len(ocrTransactions) > len(transactions))):
		transactions, warnings = ocrTransactions, ocrWarnings
	case exactErr == nil:
	case ocrErr == nil:
		transactions, warnings = ocrTransactions, ocrWarnings
	default:
		return edocuenta.ParseResult{}, exactErr
	}

	return edocuenta.ParseResult{
		Statement: edocuenta.Statement{
			Bank:          edocuenta.BankHSBC,
			AccountNumber: account,
			Currency:      edocuenta.CurrencyMXN,
			PeriodStart:   periodStart,
			PeriodEnd:     periodEnd,
			AccountClass:  edocuenta.AccountClassLiability,
			Transactions:  transactions,
		},
		Warnings: warnings,
	}, nil
}

func parseCardTransactions(text string) ([]edocuenta.Transaction, []string, error) {
	var (
		transactions []edocuenta.Transaction
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
		return nil, warnings, fmt.Errorf("hsbc: no transactions found")
	}

	return transactions, warnings, nil
}

func parseOCRCardTransactions(text string, periodStart, periodEnd time.Time) ([]edocuenta.Transaction, []string, error) {
	lines := strings.Split(text, "\n")
	transactions := make([]edocuenta.Transaction, 0)
	warnings := make([]string, 0)
	carryDescription := make([]string, 0, 2)
	scanning := false
	start := firstCardSectionStart(lines)
	if start < 0 {
		start = 0
	}

	for i := start; i < len(lines); i++ {
		line := normalizeOCRCardLine(lines[i])
		if line == "" {
			continue
		}

		upper := strings.ToUpper(line)
		if strings.Contains(upper, "DESGLOSE DE MOVIMIENTOS") || strings.Contains(upper, "CARGOS, ABONOS Y COMPRAS REGULARES") {
			continue
		}
		if isCardTerminator(upper) {
			break
		}
		if isCardHeader(upper) {
			continue
		}

		transaction, ok, err := parseOCRCardTransaction(line, carryDescription, periodStart, periodEnd)
		if ok {
			if err != nil {
				warnings = append(warnings, err.Error()+": "+line)
			} else {
				transactions = append(transactions, transaction)
				scanning = true
			}
			carryDescription = carryDescription[:0]
			continue
		}

		if looksLikeOCRDateLine(line, periodStart, periodEnd) {
			transaction, endIdx, ok, err := parseSplitOCRCardTransaction(lines, i, carryDescription, periodStart, periodEnd)
			if ok {
				if err != nil {
					warnings = append(warnings, err.Error()+": "+line)
				} else {
					transactions = append(transactions, transaction)
					scanning = true
				}
				carryDescription = carryDescription[:0]
				i = endIdx
				continue
			}
		}

		if shouldCarryCardDescription(upper, scanning) {
			carryDescription = append(carryDescription, line)
			if len(carryDescription) > 2 {
				carryDescription = carryDescription[len(carryDescription)-2:]
			}
			continue
		}

		carryDescription = carryDescription[:0]
	}

	if len(transactions) == 0 {
		return nil, warnings, fmt.Errorf("hsbc: no transactions found")
	}

	return transactions, warnings, nil
}

func parseSplitOCRCardTransaction(lines []string, start int, carry []string, periodStart, periodEnd time.Time) (edocuenta.Transaction, int, bool, error) {
	operationLine := normalizeOCRCardLine(lines[start])
	if !looksLikeOCRDateLine(operationLine, periodStart, periodEnd) {
		return edocuenta.Transaction{}, start, false, nil
	}

	_, operationErr := parseOCRCardDate(operationLine, periodStart, periodEnd)
	if operationErr != nil {
		return edocuenta.Transaction{}, start, true, fmt.Errorf("hsbc: invalid ocr operation date")
	}

	postedIdx, postedLine, ok := nextOCRCardContentLine(lines, start+1)
	if !ok || !looksLikeOCRDateLine(postedLine, periodStart, periodEnd) {
		return edocuenta.Transaction{}, start, false, nil
	}

	postedAt, err := parseOCRCardDate(postedLine, periodStart, periodEnd)
	if err != nil {
		return edocuenta.Transaction{}, postedIdx, true, fmt.Errorf("hsbc: invalid ocr posting date")
	}

	descriptionParts := append([]string{}, carry...)
	sign := ""
	endIdx := postedIdx

	for i := postedIdx + 1; i < len(lines); i++ {
		line := normalizeOCRCardLine(lines[i])
		if line == "" {
			continue
		}

		upper := strings.ToUpper(line)
		if isCardTerminator(upper) {
			break
		}
		if isCardHeader(upper) {
			continue
		}
		if looksLikeStandaloneSign(line) {
			sign = firstMovementSign(line)
			endIdx = i
			continue
		}
		if amountSign, amountText, ok := extractStandaloneOCRAmount(line); ok {
			if sign == "" {
				sign = amountSign
			}

			description := normalize.CollapseWhitespace(strings.Join(descriptionParts, " "))
			if description == "" {
				return edocuenta.Transaction{}, i, true, fmt.Errorf("hsbc: missing ocr description")
			}

			amountCents, err := parseOCRAmountWithFX(description, amountText)
			if err != nil {
				return edocuenta.Transaction{}, i, true, err
			}
			if sign == "" {
				sign = inferOCRMovementSign(description)
			}

			rawParts := append(append([]string{}, carry...), operationLine, postedLine)
			rawParts = append(rawParts, descriptionParts[len(carry):]...)
			rawParts = append(rawParts, line)
			_ = rawParts

			return edocuenta.Transaction{
				PostedAt:    postedAt,
				Description: description,
				Direction:   movementType(sign),
				AmountCents: amountCents,
			}, i, true, nil
		}
		if looksLikeOCRDateLine(line, periodStart, periodEnd) {
			return edocuenta.Transaction{}, i - 1, true, fmt.Errorf("hsbc: incomplete ocr transaction")
		}

		descriptionParts = append(descriptionParts, line)
		endIdx = i
	}

	return edocuenta.Transaction{}, endIdx, true, fmt.Errorf("hsbc: incomplete ocr transaction")
}

func parseOCRCardTransaction(line string, carry []string, periodStart, periodEnd time.Time) (edocuenta.Transaction, bool, error) {
	match := cardOCRLinePattern.FindStringSubmatch(line)
	if len(match) != 4 {
		return edocuenta.Transaction{}, false, nil
	}

	if _, err := parseOCRCardDate(match[1], periodStart, periodEnd); err != nil {
		return edocuenta.Transaction{}, true, fmt.Errorf("hsbc: invalid ocr operation date")
	}

	postedAt, err := parseOCRCardDate(match[2], periodStart, periodEnd)
	if err != nil {
		return edocuenta.Transaction{}, true, fmt.Errorf("hsbc: invalid ocr posting date")
	}

	description, sign, amountCents, err := parseOCRCardDetails(match[3], carry)
	if err != nil {
		return edocuenta.Transaction{}, true, err
	}

	return edocuenta.Transaction{
		PostedAt:    postedAt,
		Description: description,
		Direction:   movementType(sign),
		AmountCents: amountCents,
	}, true, nil
}

func parseOCRCardDetails(remainder string, carry []string) (string, string, int64, error) {
	descriptionPart, sign, amountText, amountFound := splitOCRAmount(remainder)
	description := normalize.CollapseWhitespace(strings.Join(append(append([]string{}, carry...), descriptionPart), " "))
	if description == "" {
		description = normalize.CollapseWhitespace(strings.Join(carry, " "))
	}

	amountCents, err := parseOCRMoney(amountText)
	if !amountFound || err != nil {
		amountCents = 0
	}

	if fxAmount, ok := parseOCRForeignAmount(remainder); ok {
		if amountCents == 0 || absInt64(amountCents-fxAmount) > 100 {
			amountCents = fxAmount
		}
	}

	if amountCents == 0 {
		return "", "", 0, fmt.Errorf("hsbc: invalid ocr amount")
	}

	if sign == "" {
		sign = inferOCRMovementSign(description)
	}

	return description, sign, amountCents, nil
}

func parseOCRAmountWithFX(description, amountText string) (int64, error) {
	amountCents, err := parseOCRMoney(amountText)
	if err != nil {
		amountCents = 0
	}

	if fxAmount, ok := parseOCRForeignAmount(description); ok {
		if amountCents == 0 || absInt64(amountCents-fxAmount) > 100 {
			amountCents = fxAmount
		}
	}

	if amountCents == 0 {
		return 0, fmt.Errorf("hsbc: invalid ocr amount")
	}

	return amountCents, nil
}

func splitOCRAmount(line string) (string, string, string, bool) {
	line = normalize.NormalizeOCRAmountLine(line)

	indices := cardOCRAmountPattern.FindStringSubmatchIndex(line)
	if len(indices) == 0 {
		return normalize.CollapseWhitespace(line), "", "", false
	}

	description := normalize.CollapseWhitespace(line[:indices[0]])
	sign := ""
	if indices[2] >= 0 && indices[3] >= 0 {
		sign = line[indices[2]:indices[3]]
	}

	return description, sign, line[indices[4]:indices[5]], true
}

func parseOCRMoney(value string) (int64, error) {
	return normalize.ParseOCRMoneyToCents(value)
}

func extractStandaloneOCRAmount(line string) (string, string, bool) {
	description, sign, amount, ok := splitOCRAmount(line)
	if !ok || description != "" {
		return "", "", false
	}
	return sign, amount, true
}

func parseOCRForeignAmount(line string) (int64, bool) {
	match := cardOCRFXPattern.FindStringSubmatch(line)
	if len(match) != 3 {
		return 0, false
	}

	usdAmount, err := parseOCRDecimal(match[1])
	if err != nil {
		return 0, false
	}

	exchangeRate, err := parseOCRDecimal(match[2])
	if err != nil {
		return 0, false
	}

	return int64(math.Round(usdAmount * exchangeRate * 100)), true
}

func parseOCRDecimal(value string) (float64, error) {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return 0, fmt.Errorf("empty decimal")
	}

	if strings.Count(clean, ",") == 1 && !strings.Contains(clean, ".") {
		clean = strings.ReplaceAll(clean, ",", ".")
	}

	clean = strings.ReplaceAll(clean, ",", "")
	return strconv.ParseFloat(clean, 64)
}

func parseOCRCardDate(value string, periodStart, periodEnd time.Time) (time.Time, error) {
	parts := strings.Split(strings.TrimSpace(strings.Trim(value, ".,;:")), "-")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("hsbc: invalid date %q", value)
	}

	month, ok := normalize.ParseSpanishMonth(parts[1])
	if !ok {
		return time.Time{}, fmt.Errorf("hsbc: invalid month %q", parts[1])
	}

	day, err := normalizeOCRDay(parts[0])
	if err != nil {
		return time.Time{}, err
	}

	year, err := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil {
		return time.Time{}, err
	}

	year = chooseOCRYear(year, month, day, periodStart, periodEnd)
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), nil
}

func normalizeOCRDay(value string) (int, error) {
	token := strings.TrimSpace(value)
	if token == "" {
		return 0, fmt.Errorf("hsbc: empty day token")
	}
	if len(token) > 2 {
		token = token[len(token)-2:]
	}

	options := make([][]rune, 0, len(token))
	for _, r := range token {
		candidates := ocrDigitCandidates(r)
		if len(candidates) == 0 {
			return 0, fmt.Errorf("hsbc: invalid day token %q", value)
		}
		options = append(options, candidates)
	}

	bestValue := -1
	bestScore := int(^uint(0) >> 1)
	var walk func(int, []rune, int)
	walk = func(idx int, current []rune, score int) {
		if idx == len(options) {
			candidate := string(current)
			if len(candidate) == 1 {
				candidate = "0" + candidate
			}

			day, err := strconv.Atoi(candidate)
			if err != nil || day < 1 || day > 31 {
				return
			}
			if score < bestScore {
				bestValue = day
				bestScore = score
			}
			return
		}

		for pos, candidate := range options[idx] {
			walk(idx+1, append(current, candidate), score+pos)
		}
	}

	walk(0, nil, 0)
	if bestValue == -1 {
		return 0, fmt.Errorf("hsbc: invalid day token %q", value)
	}

	return bestValue, nil
}

func ocrDigitCandidates(r rune) []rune {
	switch r {
	case '0', '1', '2', '3', '5', '6', '8':
		return []rune{r}
	case '4':
		return []rune{'4', '1'}
	case '7':
		return []rune{'7', '1'}
	case '9':
		return []rune{'9', '0'}
	case 'O', 'o', 'Q', 'D':
		return []rune{'0'}
	case 'I', 'i', 'L', 'l', 'T':
		return []rune{'1'}
	case 'Z', 'z':
		return []rune{'2'}
	case 'S', 's':
		return []rune{'5'}
	case 'B':
		return []rune{'8', '0'}
	default:
		return nil
	}
}

func chooseOCRYear(rawYear int, month time.Month, day int, periodStart, periodEnd time.Time) int {
	windowStart := periodStart.AddDate(0, 0, -31)
	windowEnd := periodEnd.AddDate(0, 0, 31)
	bestYear := rawYear
	bestDistance := time.Duration(1<<63 - 1)

	for _, year := range uniqueInts(rawYear, periodStart.Year(), periodEnd.Year()) {
		candidate := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		distance := distanceToRange(candidate, windowStart, windowEnd)
		if year == rawYear {
			distance += time.Millisecond
		}
		if distance < bestDistance {
			bestDistance = distance
			bestYear = year
		}
	}

	return bestYear
}

func distanceToRange(value, start, end time.Time) time.Duration {
	if value.Before(start) {
		return start.Sub(value)
	}
	if value.After(end) {
		return value.Sub(end)
	}
	return 0
}

func uniqueInts(values ...int) []int {
	seen := make(map[int]struct{}, len(values))
	result := make([]int, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func normalizeOCRCardLine(raw string) string {
	return normalize.NormalizeOCRLine(raw)
}

func nextOCRCardContentLine(lines []string, start int) (int, string, bool) {
	for i := start; i < len(lines); i++ {
		line := normalizeOCRCardLine(lines[i])
		if line == "" {
			continue
		}
		return i, line, true
	}
	return -1, "", false
}

func firstCardSectionStart(lines []string) int {
	for i, raw := range lines {
		upper := strings.ToUpper(normalizeOCRCardLine(raw))
		if strings.Contains(upper, "DESGLOSE DE MOVIMIENTOS") {
			return i
		}
	}

	return -1
}

func isCardHeader(line string) bool {
	return strings.HasPrefix(line, "HSBC") ||
		strings.HasPrefix(line, "PAGINA ") ||
		strings.Contains(line, "NUMERO DE CUENTA") ||
		strings.Contains(line, "CARGOS, ABONOS Y COMPRAS REGULARES") ||
		strings.Contains(line, "FECHA DE LA") ||
		strings.Contains(line, "FECHA DE CARGO") ||
		strings.Contains(line, "DESCRIPCION DEL MOVIMIENTO") ||
		strings.Contains(line, "IV. MONTO") ||
		line == "OPERACION" ||
		line == "NOTAS:" ||
		strings.HasPrefix(line, "VER NOTAS") ||
		strings.Contains(line, "TARJETA ADICIONAL") ||
		strings.Contains(line, "TARJETA TITULAR") ||
		strings.HasPrefix(line, "TOTAL CARGOS") ||
		strings.HasPrefix(line, "TOTAL ABONOS")
}

func isCardTerminator(line string) bool {
	return line == "ATENCION DE QUEJAS" || line == "NOTAS ACLARATORIAS"
}

func looksLikeOCRDateLine(line string, periodStart, periodEnd time.Time) bool {
	_, err := parseOCRCardDate(line, periodStart, periodEnd)
	return err == nil
}

func looksLikeStandaloneSign(line string) bool {
	return line == "+" || line == "-"
}

func firstMovementSign(line string) string {
	if strings.Contains(line, "-") {
		return "-"
	}
	return "+"
}

func shouldCarryCardDescription(line string, scanning bool) bool {
	if strings.Contains(line, "MONEDA EXTRANJERA") {
		return false
	}
	if strings.Contains(line, "NUMERO DE CUENTA") || strings.Contains(line, "PAGINA ") {
		return false
	}
	if !scanning && !strings.Contains(line, "OPENAI") {
		return false
	}

	return strings.IndexFunc(line, func(r rune) bool {
		return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
	}) != -1
}

func inferOCRMovementSign(description string) string {
	upper := strings.ToUpper(description)
	if strings.Contains(upper, "SU PAGO GRACIAS") {
		return "-"
	}

	return "+"
}

func looksLikeOCRCardText(text string) bool {
	return strings.Contains(text, "+]$") ||
		strings.Contains(text, "+ [$") ||
		strings.Contains(text, "+1$") ||
		strings.Contains(text, "__") ||
		strings.Contains(text, "MONEDA EXTRANJERA") ||
		strings.Contains(text, " O7-") ||
		strings.Contains(text, " T1-") ||
		strings.Contains(text, " 42-")
}

func absInt64(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
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

func parseFullTransaction(line string) (edocuenta.Transaction, bool, error) {
	match := cardFullTxPattern.FindStringSubmatch(line)
	if len(match) != 6 {
		return edocuenta.Transaction{}, false, nil
	}

	postedAt, err := parseSpanishDate(match[2])
	if err != nil {
		return edocuenta.Transaction{}, true, fmt.Errorf("hsbc: invalid posting date")
	}

	amountCents, err := normalize.ParseMoneyToCents(match[5])
	if err != nil {
		return edocuenta.Transaction{}, true, fmt.Errorf("hsbc: invalid amount")
	}

	return edocuenta.Transaction{
		PostedAt:    postedAt,
		Description: normalize.CollapseWhitespace(match[3]),
		Direction:   movementType(match[4]),
		AmountCents: amountCents,
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

func completePending(pending pendingTransaction, line string) (edocuenta.Transaction, bool, error) {
	match := cardAmountPattern.FindStringSubmatch(line)
	if len(match) != 4 {
		return edocuenta.Transaction{}, false, nil
	}

	amountCents, err := normalize.ParseMoneyToCents(match[3])
	if err != nil {
		return edocuenta.Transaction{}, true, fmt.Errorf("hsbc: invalid amount")
	}

	description := pending.description
	continuation := normalize.CollapseWhitespace(match[1])
	if continuation != "" {
		description = normalize.CollapseWhitespace(description + " " + continuation)
	}

	return edocuenta.Transaction{
		PostedAt:    pending.postedAt,
		Description: description,
		Direction:   movementType(match[2]),
		AmountCents: amountCents,
	}, true, nil
}

func parseSpanishDate(value string) (time.Time, error) {
	parsed, err := normalize.ParseDateDDMonYYYYSpanish(value)
	if err != nil {
		return time.Time{}, fmt.Errorf("hsbc: %w", err)
	}
	return parsed, nil
}

func movementType(sign string) edocuenta.TransactionDirection {
	if sign == "-" {
		return edocuenta.TransactionDirectionCredit
	}
	return edocuenta.TransactionDirectionDebit
}

func hsbcDetectionScore(text string) int {
	upper := strings.ToUpper(text)
	if !strings.Contains(upper, "HSBC") {
		return 0
	}

	score := 3
	if cardAccountPattern.MatchString(text) || flexibleAccountPattern.MatchString(text) {
		score += 2
	}
	if cardPeriodPattern.MatchString(text) || flexiblePeriodPattern.MatchString(text) {
		score += 3
	}
	if strings.Contains(upper, "CUENTA FLEXIBLE") {
		score += 2
	}
	if strings.Contains(upper, "DESGLOSE DE MOVIMIENTOS") || strings.Contains(upper, "DETALLE MOVIMIENTOS CUENTA FLEXIBLE") {
		score += 2
	}
	if hasCardTransactions(text) || hasFlexibleHeaders(text) {
		score++
	}

	if score < hsbcDetectThreshold {
		return 0
	}

	return score
}

func hasCardTransactions(text string) bool {
	for _, rawLine := range strings.Split(text, "\n") {
		line := normalize.CollapseWhitespace(rawLine)
		if cardFullTxPattern.MatchString(line) || cardOpenTxPattern.MatchString(line) {
			return true
		}
	}

	return false
}

func hasFlexibleHeaders(text string) bool {
	for _, rawLine := range strings.Split(text, "\n") {
		if looksLikeFlexibleHeader(strings.TrimSpace(rawLine)) {
			return true
		}
	}

	return false
}

func looksLikeContinuation(line string) bool {
	if line == "" {
		return false
	}

	return !cardOpenTxPattern.MatchString(line)
}

func parseFlexibleResult(text string) (edocuenta.ParseResult, error) {
	account, err := parseFlexibleAccount(text)
	if err != nil {
		return edocuenta.ParseResult{}, err
	}

	periodStart, periodEnd, err := parseFlexiblePeriod(text)
	if err != nil {
		return edocuenta.ParseResult{}, err
	}

	initialBalance, err := parseFlexibleInitialBalance(text)
	if err != nil {
		return edocuenta.ParseResult{}, err
	}

	transactions, warnings, err := parseFlexibleTransactions(text, periodStart, periodEnd, initialBalance)
	if err != nil {
		return edocuenta.ParseResult{}, err
	}

	return edocuenta.ParseResult{
		Statement: edocuenta.Statement{
			Bank:          edocuenta.BankHSBC,
			AccountNumber: account,
			Currency:      edocuenta.CurrencyMXN,
			PeriodStart:   periodStart,
			PeriodEnd:     periodEnd,
			AccountClass:  edocuenta.AccountClassAsset,
			Summary:       buildFlexibleSummary(text, initialBalance),
			Transactions:  transactions,
		},
		Warnings: warnings,
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

	start, err := parseFlexiblePeriodDate(match[1])
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	end, err := parseFlexiblePeriodDate(match[2])
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	return start, end, nil
}

func parseFlexibleInitialBalance(text string) (int64, error) {
	match := flexibleInitialPattern.FindStringSubmatch(text)
	if len(match) != 2 {
		lines := strings.Split(text, "\n")
		for i, rawLine := range lines {
			line := normalize.CollapseWhitespace(rawLine)
			if !strings.Contains(strings.ToUpper(line), "SALDO INICIAL DEL") {
				continue
			}

			window := line
			nonEmpty := 0
			for j := i + 1; j < len(lines) && nonEmpty < 6; j++ {
				next := normalize.CollapseWhitespace(lines[j])
				if next == "" {
					continue
				}

				window = normalize.CollapseWhitespace(window + " " + next)
				nonEmpty++
			}

			money := flexibleMoneyPattern.FindStringSubmatch(window)
			if len(money) == 2 {
				return normalize.ParseMoneyToCents(money[1])
			}
		}

		return 0, fmt.Errorf("hsbc: initial balance not found")
	}

	return normalize.ParseMoneyToCents(match[1])
}

func parseFlexibleClosingBalance(text string) (*int64, bool) {
	match := flexibleClosingPattern.FindStringSubmatch(text)
	if len(match) != 2 {
		return nil, false
	}

	value, err := normalize.ParseMoneyToCents(match[1])
	if err != nil {
		return nil, false
	}

	return &value, true
}

func buildFlexibleSummary(text string, initialBalance int64) *edocuenta.StatementSummary {
	summary := &edocuenta.StatementSummary{
		OpeningBalanceCents: int64Ptr(initialBalance),
	}

	if closingBalance, ok := parseFlexibleClosingBalance(text); ok {
		summary.ClosingBalanceCents = closingBalance
	}

	return summary
}

func int64Ptr(value int64) *int64 {
	return &value
}

func parseFlexibleTransactions(text string, periodStart, periodEnd time.Time, initialBalance int64) ([]edocuenta.Transaction, []string, error) {
	lines := strings.Split(text, "\n")
	var (
		transactions []edocuenta.Transaction
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

		if isFlexibleSectionEnd(line) {
			break
		}

		if !looksLikeFlexibleHeaderAt(lines, i) {
			continue
		}

		header := line
		description, reference := parseFlexibleDescriptionAndReference(header[2:])
		postedAt, err := inferFlexibleDate(strings.TrimSpace(header[:2]), periodStart, periodEnd)
		if err != nil {
			warnings = append(warnings, "invalid date: "+line)
			continue
		}

		var (
			serials         []string
			amountCents     int64
			balanceCents    int64
			amountFound     bool
			stoppedByMarker bool
			ignored         bool
		)
		for j := i + 1; j < len(lines); j++ {
			next := strings.TrimSpace(lines[j])
			if next == "" {
				continue
			}
			if isFlexibleSectionEnd(next) {
				stoppedByMarker = true
				i = j - 1
				break
			}
			if amount, balance, endIdx, ok, err := parseFlexibleAmountsAt(lines, j); ok {
				if err != nil {
					warnings = append(warnings, "invalid amount line: "+next)
					i = endIdx
					ignored = true
					break
				}
				amountCents = amount
				balanceCents = balance
				amountFound = true
				i = endIdx
				break
			}
			if isFlexibleSerial(next) {
				serials = append(serials, next)
				continue
			}
			if looksLikeFlexibleHeaderAt(lines, j) {
				if !isFlexibleInformationalLine(line) {
					warnings = append(warnings, "line ignored: "+line)
				}
				ignored = true
				i = j - 1
				break
			}
			description = normalize.CollapseWhitespace(description + " " + next)
		}

		if stoppedByMarker {
			break
		}

		if !amountFound {
			if ignored {
				continue
			}
			if !isFlexibleInformationalLine(line) {
				warnings = append(warnings, "line ignored: "+line)
			}
			continue
		}

		txType := inferFlexibleMovementType(prevBalance, amountCents, balanceCents, description)
		balanceCopy := balanceCents
		txReference := reference
		for _, serial := range serials {
			if txReference != "" {
				txReference = txReference + "/" + serial
			} else {
				txReference = serial
			}
		}

		transactions = append(transactions, edocuenta.Transaction{
			PostedAt:     postedAt,
			Description:  description,
			Reference:    txReference,
			Direction:    txType,
			AmountCents:  amountCents,
			BalanceCents: &balanceCopy,
		})

		prevBalance = balanceCents
	}

	if len(transactions) == 0 {
		return nil, warnings, fmt.Errorf("hsbc: no transactions found")
	}

	return transactions, warnings, nil
}

func isFlexibleInformationalLine(line string) bool {
	normalized := strings.ToUpper(normalize.CollapseWhitespace(line))
	return strings.Contains(normalized, "APERTURA DE CUENTA")
}

func parseFlexibleAmountsAt(lines []string, start int) (int64, int64, int, bool, error) {
	line := strings.TrimSpace(lines[start])
	if match := flexibleAmountPattern.FindStringSubmatch(line); len(match) == 3 {
		amount, err := normalize.ParseMoneyToCents(match[1])
		if err != nil {
			return 0, 0, start, true, err
		}
		balance, err := normalize.ParseMoneyToCents(match[2])
		if err != nil {
			return 0, 0, start, true, err
		}
		return amount, balance, start, true, nil
	}

	first := flexibleSingleAmount.FindStringSubmatch(line)
	if len(first) != 2 {
		return 0, 0, start, false, nil
	}

	nextIdx := -1
	var nextLine string
	for i := start + 1; i < len(lines); i++ {
		next := strings.TrimSpace(lines[i])
		if next == "" {
			continue
		}
		nextIdx = i
		nextLine = next
		break
	}
	if nextIdx == -1 {
		return 0, 0, start, false, nil
	}

	second := flexibleSingleAmount.FindStringSubmatch(nextLine)
	if len(second) != 2 {
		return 0, 0, start, false, nil
	}

	amount, err := normalize.ParseMoneyToCents(first[1])
	if err != nil {
		return 0, 0, nextIdx, true, err
	}
	balance, err := normalize.ParseMoneyToCents(second[1])
	if err != nil {
		return 0, 0, nextIdx, true, err
	}

	return amount, balance, nextIdx, true, nil
}

func looksLikeFlexibleHeader(line string) bool {
	line = strings.TrimSpace(line)
	if len(line) < 3 {
		return false
	}

	if !looksLikeFlexibleDayPrefix(line) {
		return false
	}
	if line[2] >= '0' && line[2] <= '9' {
		return false
	}

	return strings.IndexFunc(line[2:], func(r rune) bool {
		return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
	}) != -1
}

func looksLikeFlexibleHeaderAt(lines []string, idx int) bool {
	line := strings.TrimSpace(lines[idx])
	if looksLikeFlexibleHeader(line) {
		return true
	}
	if !looksLikeFlexibleDayPrefix(line) {
		return false
	}

	nonEmpty := 0
	for j := idx + 1; j < len(lines) && nonEmpty < 6; j++ {
		next := strings.TrimSpace(lines[j])
		if next == "" {
			continue
		}
		if isFlexibleSectionEnd(next) {
			return false
		}

		nonEmpty++
		if _, _, _, ok, _ := parseFlexibleAmountsAt(lines, j); ok {
			return true
		}
		if isFlexibleSerial(next) {
			continue
		}
		return false
	}

	return false
}

func looksLikeFlexibleDayPrefix(line string) bool {
	if len(line) < 2 {
		return false
	}
	if line[0] < '0' || line[0] > '9' || line[1] < '0' || line[1] > '9' {
		return false
	}

	day, err := strconv.Atoi(line[:2])
	return err == nil && day >= 1 && day <= 31
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

func isFlexibleSectionEnd(line string) bool {
	upper := strings.ToUpper(strings.TrimSpace(line))
	switch {
	case strings.HasPrefix(upper, "SALDO INICIAL $"),
		strings.HasPrefix(upper, "SALDO FINAL $"),
		strings.HasPrefix(upper, "INFORMACIÓNSPEI"),
		strings.HasPrefix(upper, "INFORMACIONSPEI"),
		strings.HasPrefix(upper, "INFORMACIÓN SPEI"),
		strings.HasPrefix(upper, "INFORMACION SPEI"),
		strings.HasPrefix(upper, "INFORMACIÓN CODI"),
		strings.HasPrefix(upper, "INFORMACION CODI"),
		strings.HasPrefix(upper, "ACLARACIONES"),
		strings.HasPrefix(upper, "PROMOCIONES"),
		strings.HasPrefix(upper, "MENSAJES IMPORTANTES"),
		strings.HasPrefix(upper, "INFORMACIÓN GENERAL"),
		strings.HasPrefix(upper, "INFORMACION GENERAL"),
		strings.HasPrefix(upper, "CONTÁCTANOS"),
		strings.HasPrefix(upper, "CONTACTANOS"):
		return true
	default:
		return false
	}
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

func inferFlexibleMovementType(prevBalance, amountCents, balanceCents int64, description string) edocuenta.TransactionDirection {
	switch {
	case prevBalance+amountCents == balanceCents:
		return edocuenta.TransactionDirectionCredit
	case prevBalance-amountCents == balanceCents:
		return edocuenta.TransactionDirectionDebit
	case strings.Contains(strings.ToUpper(description), "PAGO DE TARJETA"):
		return edocuenta.TransactionDirectionDebit
	default:
		return edocuenta.TransactionDirectionCredit
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

func parseFlexiblePeriodDate(value string) (time.Time, error) {
	if strings.Contains(value, "/") {
		return parseSlashDate(value)
	}

	return parseCompactDate(value)
}

func parseSlashDate(value string) (time.Time, error) {
	parts := strings.Split(strings.ReplaceAll(strings.TrimSpace(value), " ", ""), "/")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("hsbc: invalid slash date %q", value)
	}

	day, err := strconv.Atoi(parts[0])
	if err != nil {
		return time.Time{}, err
	}
	month, err := strconv.Atoi(parts[1])
	if err != nil {
		return time.Time{}, err
	}
	year, err := strconv.Atoi(parts[2])
	if err != nil {
		return time.Time{}, err
	}

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
}
