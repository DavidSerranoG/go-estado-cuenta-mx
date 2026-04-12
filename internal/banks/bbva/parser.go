package bbva

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	edocuenta "github.com/DavidSerranoG/go-estado-cuenta-mx"
	"github.com/DavidSerranoG/go-estado-cuenta-mx/internal/normalize"
)

const (
	bbvaFullDatePattern  = `[0-9]{2}/[0-9]{2}/[0-9]{4}`
	bbvaShortDatePattern = `[0-9]{2}\s*/\s*(?:ENE|FEB|MAR|ABR|MAY|JUN|JUL|AGO|SEP|OCT|NOV|DIC)`
	bbvaDetectThreshold  = 6
)

var (
	accountPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)cuenta:\s*([0-9][0-9 ]{7,20})`),
		regexp.MustCompile(`(?i)cuenta\s+([0-9][0-9 ]{7,20})`),
		regexp.MustCompile(`(?i)no\.?\s*de\s*cuenta\s*([0-9][0-9 ]{7,20})`),
		regexp.MustCompile(`(?i)n[úu]mero\s*de\s*cuenta:?\s*([0-9][0-9 ]{7,20})`),
	}
	clabePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:no\.?\s*)?cuenta\s*clabe\s*:?\s*((?:[0-9]\s*){18})`),
		regexp.MustCompile(`(?i)clabe\s*interbancaria\s*:?\s*((?:[0-9]\s*){18})`),
	}
	periodPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)periodo\s*:?\s*(?:del\s*)?(` + bbvaFullDatePattern + `)\s*(?:-|a|al)\s*(` + bbvaFullDatePattern + `)`),
	}
	openingBalancePattern = regexp.MustCompile(`(?i)saldo anterior(?:\s*\([+-]\))?\s*([0-9,]+\.\d{2})`)
	closingBalancePattern = regexp.MustCompile(`(?i)saldo final(?:\s*\([+-]\))?\s*([0-9,]+\.\d{2})`)
	summaryPattern        = regexp.MustCompile(`(?is)total importe cargos\s*([0-9,]+\.\d{2})\s*total movimientos cargos\s*([0-9]+)\s*total importe abonos\s*([0-9,]+\.\d{2})\s*total movimientos abonos\s*([0-9]+)`)
	legacyTxPattern       = regexp.MustCompile(`(?i)^([0-9]{2}/[0-9]{2}/[0-9]{4})\s+(.+?)\s+(ABONO|CARGO)\s+([0-9,]+\.\d{2})\s+([0-9,]+\.\d{2})$`)
	realTxStartPattern    = regexp.MustCompile(`(` + bbvaShortDatePattern + `)\s*(` + bbvaShortDatePattern + `)`)
	realTxPattern         = regexp.MustCompile(`^(` + bbvaShortDatePattern + `)\s*(` + bbvaShortDatePattern + `)\s*(.+?)([0-9,]+\.\d{2})([0-9,]+\.\d{2})([0-9,]+\.\d{2})\s*(.*)$`)
	realTxAmountOnly      = regexp.MustCompile(`^(` + bbvaShortDatePattern + `)\s*(` + bbvaShortDatePattern + `)\s*(.+?)([0-9,]+\.\d{2})\s*(.*)$`)
)

// Parser parses BBVA bank statements.
type Parser struct{}

// New returns a new BBVA parser.
func New() Parser {
	return Parser{}
}

// Bank returns the canonical bank identifier.
func (Parser) Bank() string {
	return string(edocuenta.BankBBVA)
}

// DetectionScore returns a structural confidence score for BBVA layouts.
func (Parser) DetectionScore(text string) int {
	return bbvaDetectionScore(text)
}

// Layout returns the normalized BBVA layout identifier.
func (Parser) Layout(text string) string {
	text = normalize.NormalizeExtractedText(text)
	if looksLikeCardStatement(text) {
		return "card"
	}
	return "account"
}

// CanParse checks whether the extracted text looks like BBVA.
func (Parser) CanParse(text string) bool {
	return bbvaDetectionScore(text) > 0
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

	if looksLikeCardStatement(text) {
		return parseCardResult(text)
	}

	account, err := parseAccount(text)
	if err != nil {
		return edocuenta.ParseResult{}, err
	}

	periodStart, periodEnd, err := parsePeriod(text)
	if err != nil {
		return edocuenta.ParseResult{}, err
	}

	var (
		transactions []edocuenta.Transaction
		warnings     []string
	)

	transactions, warnings, err = parseTransactions(text, periodStart, periodEnd)
	if err != nil {
		return edocuenta.ParseResult{}, err
	}

	if len(transactions) == 0 {
		return edocuenta.ParseResult{}, fmt.Errorf("bbva: no transactions found")
	}

	return edocuenta.ParseResult{
		Statement: edocuenta.Statement{
			Bank:          edocuenta.BankBBVA,
			AccountNumber: account,
			Currency:      parseCurrency(text),
			PeriodStart:   periodStart,
			PeriodEnd:     periodEnd,
			AccountClass:  edocuenta.AccountClassAsset,
			Summary:       buildAccountSummary(text),
			Transactions:  transactions,
		},
		Warnings: warnings,
	}, nil
}

func parseAccount(text string) (string, error) {
	for _, pattern := range accountPatterns {
		match := pattern.FindStringSubmatch(text)
		if len(match) != 2 {
			continue
		}

		account := normalize.DigitsOnly(match[1])
		if len(account) >= 8 {
			return account, nil
		}
	}

	for _, pattern := range clabePatterns {
		match := pattern.FindStringSubmatch(text)
		if len(match) != 2 {
			continue
		}

		clabe := normalize.DigitsOnly(match[1])
		if len(clabe) != 18 {
			continue
		}

		return clabe[7:17], nil
	}

	return "", fmt.Errorf("bbva: account number not found")
}

func parsePeriod(text string) (time.Time, time.Time, error) {
	for _, pattern := range periodPatterns {
		match := pattern.FindStringSubmatch(text)
		if len(match) != 3 {
			continue
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

	return time.Time{}, time.Time{}, fmt.Errorf("bbva: period not found")
}

func parseCurrency(text string) edocuenta.Currency {
	upper := strings.ToUpper(text)
	switch {
	case strings.Contains(upper, "MONEDA DOLARES"),
		strings.Contains(upper, "MONEDA DÓLARES"),
		strings.Contains(upper, "LIBRETÓN DÓLARES"),
		strings.Contains(upper, "LIBRETON DOLARES"),
		strings.Contains(upper, "CUENTA EN DOLARES"),
		strings.Contains(upper, "CUENTA EN DÓLARES"):
		return edocuenta.CurrencyUSD
	case strings.Contains(upper, "MONEDA NACIONAL"),
		strings.Contains(upper, "M.N."),
		strings.Contains(upper, "PESOS MEXICANOS"),
		strings.Contains(upper, "MXN"):
		return edocuenta.CurrencyMXN
	default:
		return edocuenta.CurrencyMXN
	}
}

func bbvaDetectionScore(text string) int {
	upper := strings.ToUpper(text)
	if !strings.Contains(upper, "BBVA") && !strings.Contains(upper, "BANCOMER") {
		return 0
	}

	score := 3
	score += cardDetectionScore(text)
	if matchesAnyRegexp(text, periodPatterns...) {
		score += 3
	}
	if matchesAnyRegexp(text, clabePatterns...) {
		score += 2
	}
	if matchesAnyRegexp(text, accountPatterns...) {
		score++
	}
	if strings.Contains(upper, "DETALLE DE MOVIMIENTOS") {
		score += 2
	}
	if openingBalancePattern.MatchString(text) {
		score++
	}
	if summaryPattern.MatchString(text) {
		score++
	}
	if realTxStartPattern.MatchString(text) || hasLegacyTransactions(text) {
		score += 2
	}
	if parseCurrency(text) == edocuenta.CurrencyUSD {
		score++
	}

	if score < bbvaDetectThreshold {
		return 0
	}

	return score
}

func matchesAnyRegexp(text string, patterns ...*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern != nil && pattern.MatchString(text) {
			return true
		}
	}

	return false
}

func hasLegacyTransactions(text string) bool {
	for _, rawLine := range strings.Split(text, "\n") {
		if legacyTxPattern.MatchString(normalize.CollapseWhitespace(rawLine)) {
			return true
		}
	}

	return false
}

func parseTransactions(text string, periodStart, periodEnd time.Time) ([]edocuenta.Transaction, []string, error) {
	if transactions, warnings := parseRealTransactions(text, periodStart, periodEnd); len(transactions) > 0 {
		return transactions, warnings, nil
	}

	transactions, warnings := parseLegacyTransactions(text)
	if len(transactions) > 0 {
		return transactions, warnings, nil
	}

	return nil, warnings, nil
}

func parseLegacyTransactions(text string) ([]edocuenta.Transaction, []string) {
	var (
		transactions []edocuenta.Transaction
		warnings     []string
	)

	for _, rawLine := range strings.Split(text, "\n") {
		line := normalize.CollapseWhitespace(rawLine)
		if line == "" {
			continue
		}

		match := legacyTxPattern.FindStringSubmatch(line)
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

		amountCents, err := normalize.ParseOCRMoneyToCents(match[4])
		if err != nil {
			warnings = append(warnings, "invalid amount: "+line)
			continue
		}

		balanceCents, err := normalize.ParseOCRMoneyToCents(match[5])
		if err != nil {
			warnings = append(warnings, "invalid balance: "+line)
			continue
		}

		balanceCopy := balanceCents
		transactions = append(transactions, edocuenta.Transaction{
			PostedAt:     postedAt,
			Description:  match[2],
			Direction:    legacyMovementKind(match[3]),
			AmountCents:  amountCents,
			BalanceCents: &balanceCopy,
		})
	}

	return transactions, warnings
}

func parseRealTransactions(text string, periodStart, periodEnd time.Time) ([]edocuenta.Transaction, []string) {
	section, ok := extractTransactionSection(text)
	if !ok {
		return nil, nil
	}

	openingBalance, _ := parseOpeningBalance(text)
	summary, _ := parseTransactionSummary(text)

	chunks := splitRealTransactionChunks(section)
	if len(chunks) == 0 {
		return nil, nil
	}

	rawTransactions := make([]rawTransaction, 0, len(chunks))
	warnings := make([]string, 0)

	for _, chunk := range chunks {
		rawTx, err := parseRealTransactionChunk(chunk, periodStart, periodEnd)
		if err != nil {
			warnings = append(warnings, err.Error()+": "+normalize.CollapseWhitespace(chunk))
			continue
		}
		rawTransactions = append(rawTransactions, rawTx)
	}

	if len(rawTransactions) == 0 {
		return nil, warnings
	}

	resolveTransactionKinds(rawTransactions, openingBalance, summary)

	transactions := make([]edocuenta.Transaction, 0, len(rawTransactions))
	for _, item := range rawTransactions {
		if item.kind == "" {
			warnings = append(warnings, "transaction type unresolved: "+item.rawLine)
			continue
		}

		balance := item.balanceCents
		transactions = append(transactions, edocuenta.Transaction{
			PostedAt:     item.postedAt,
			Description:  item.description,
			Reference:    item.reference,
			Direction:    item.kind,
			AmountCents:  item.amountCents,
			BalanceCents: balance,
		})
	}

	return transactions, warnings
}

type rawTransaction struct {
	postedAt     time.Time
	description  string
	reference    string
	amountCents  int64
	balanceCents *int64
	rawLine      string
	kind         edocuenta.TransactionDirection
}

type transactionSummary struct {
	cargoAmountCents int64
	cargoCount       int
	abonoAmountCents int64
	abonoCount       int
}

func extractTransactionSection(text string) (string, bool) {
	upper := strings.ToUpper(text)
	start := -1
	for _, marker := range []string{"DETALLE DE MOVIMIENTOS REALIZADOS", "DETALLE DE MOVIMIENTOS"} {
		if idx := strings.Index(upper, marker); idx != -1 {
			start = idx
			break
		}
	}
	if start == -1 {
		return "", false
	}

	section := text[start:]
	upperSection := upper[start:]
	end := len(section)
	for _, marker := range []string{"TOTAL IMPORTE CARGOS", "TOTAL DE MOVIMIENTOS", "SALDO FINAL", "COMPORTAMIENTO"} {
		if idx := strings.Index(upperSection, marker); idx != -1 && idx < end {
			end = idx
		}
	}

	section = section[:end]
	return section, true
}

func splitRealTransactionChunks(section string) []string {
	indexes := realTxStartPattern.FindAllStringIndex(section, -1)
	if len(indexes) == 0 {
		return nil
	}

	chunks := make([]string, 0, len(indexes))
	for i, idx := range indexes {
		from := idx[0]
		to := len(section)
		if i+1 < len(indexes) {
			to = indexes[i+1][0]
		}

		chunk := strings.TrimSpace(section[from:to])
		chunk = trimNoise(chunk)
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
	}

	return chunks
}

func parseRealTransactionChunk(chunk string, periodStart, periodEnd time.Time) (rawTransaction, error) {
	if match := realTxPattern.FindStringSubmatch(chunk); len(match) == 8 {
		return buildRawTransaction(match[1], match[3], match[7], match[4], match[5], chunk, periodStart, periodEnd)
	}

	if match := realTxAmountOnly.FindStringSubmatch(chunk); len(match) == 6 {
		return buildRawTransaction(match[1], match[3], match[5], match[4], "", chunk, periodStart, periodEnd)
	}

	return rawTransaction{}, fmt.Errorf("bbva: line ignored")
}

func buildRawTransaction(dateToken, description, reference, amountValue, balanceValue, rawLine string, periodStart, periodEnd time.Time) (rawTransaction, error) {
	postedAt, err := parseShortDate(dateToken, periodStart, periodEnd)
	if err != nil {
		return rawTransaction{}, fmt.Errorf("bbva: invalid date")
	}

	amountCents, err := normalize.ParseOCRMoneyToCents(amountValue)
	if err != nil {
		return rawTransaction{}, fmt.Errorf("bbva: invalid amount")
	}

	var balanceCents *int64
	if balanceValue != "" {
		parsedBalance, err := normalize.ParseOCRMoneyToCents(balanceValue)
		if err != nil {
			return rawTransaction{}, fmt.Errorf("bbva: invalid balance")
		}
		balanceCents = &parsedBalance
	}

	return rawTransaction{
		postedAt:     postedAt,
		description:  normalize.CollapseWhitespace(description),
		reference:    normalize.CollapseWhitespace(trimNoise(reference)),
		amountCents:  amountCents,
		balanceCents: balanceCents,
		rawLine:      normalize.CollapseWhitespace(trimNoise(rawLine)),
	}, nil
}

func parseOpeningBalance(text string) (*int64, bool) {
	match := openingBalancePattern.FindStringSubmatch(text)
	if len(match) != 2 {
		return nil, false
	}

	value, err := normalize.ParseOCRMoneyToCents(match[1])
	if err != nil {
		return nil, false
	}

	return &value, true
}

func parseClosingBalance(text string) (*int64, bool) {
	match := closingBalancePattern.FindStringSubmatch(text)
	if len(match) != 2 {
		return nil, false
	}

	value, err := normalize.ParseOCRMoneyToCents(match[1])
	if err != nil {
		return nil, false
	}

	return &value, true
}

func parseTransactionSummary(text string) (*transactionSummary, bool) {
	match := summaryPattern.FindStringSubmatch(text)
	if len(match) != 5 {
		return nil, false
	}

	cargoAmountCents, err := normalize.ParseOCRMoneyToCents(match[1])
	if err != nil {
		return nil, false
	}

	cargoCount, err := strconv.Atoi(match[2])
	if err != nil {
		return nil, false
	}

	abonoAmountCents, err := normalize.ParseOCRMoneyToCents(match[3])
	if err != nil {
		return nil, false
	}

	abonoCount, err := strconv.Atoi(match[4])
	if err != nil {
		return nil, false
	}

	return &transactionSummary{
		cargoAmountCents: cargoAmountCents,
		cargoCount:       cargoCount,
		abonoAmountCents: abonoAmountCents,
		abonoCount:       abonoCount,
	}, true
}

func buildAccountSummary(text string) *edocuenta.StatementSummary {
	var (
		summary   edocuenta.StatementSummary
		populated bool
	)

	if openingBalance, ok := parseOpeningBalance(text); ok {
		summary.OpeningBalanceCents = openingBalance
		populated = true
	}

	if closingBalance, ok := parseClosingBalance(text); ok {
		summary.ClosingBalanceCents = closingBalance
		populated = true
	}

	if totals, ok := parseTransactionSummary(text); ok {
		summary.TotalDebitsCents = int64Ptr(totals.cargoAmountCents)
		summary.TotalCreditsCents = int64Ptr(totals.abonoAmountCents)
		populated = true
	}

	if !populated {
		return nil
	}

	return &summary
}

func int64Ptr(value int64) *int64 {
	return &value
}

func resolveTransactionKinds(items []rawTransaction, openingBalance *int64, summary *transactionSummary) {
	for {
		changed := false

		if resolveWithRunningBalance(items, openingBalance) {
			changed = true
		}
		if resolveWithDescriptionHints(items) {
			changed = true
		}
		if resolveWithSummary(items, summary) {
			changed = true
		}

		if !changed {
			return
		}
	}
}

func resolveWithRunningBalance(items []rawTransaction, openingBalance *int64) bool {
	if openingBalance == nil {
		return false
	}

	changed := false
	running := *openingBalance
	runningKnown := true

	for i := range items {
		if !runningKnown {
			break
		}

		item := &items[i]
		if item.kind == "" && item.balanceCents == nil {
			if inferredBalance, ok := inferMissingBalanceTransaction(items, i, running); ok {
				item.kind = inferredBalance.kind
				item.balanceCents = &inferredBalance.balance
				changed = true
			}
		}

		if item.balanceCents != nil {
			delta := *item.balanceCents - running
			switch {
			case delta < 0:
				if item.kind == "" {
					item.kind = edocuenta.TransactionDirectionDebit
					changed = true
				}
				if item.amountCents != -delta {
					item.amountCents = -delta
					changed = true
				}
			case delta > 0:
				if item.kind == "" {
					item.kind = edocuenta.TransactionDirectionCredit
					changed = true
				}
				if item.amountCents != delta {
					item.amountCents = delta
					changed = true
				}
			default:
				switch {
				case item.kind == "" && running-item.amountCents == *item.balanceCents:
					item.kind = edocuenta.TransactionDirectionDebit
					changed = true
				case item.kind == "" && running+item.amountCents == *item.balanceCents:
					item.kind = edocuenta.TransactionDirectionCredit
					changed = true
				}
			}
		}

		if item.kind == "" {
			runningKnown = false
			continue
		}

		if item.balanceCents == nil {
			nextBalance := applyAmount(running, item.amountCents, item.kind)
			item.balanceCents = &nextBalance
			changed = true
		}

		running = *item.balanceCents
	}

	return changed
}

type inferredBalanceTransaction struct {
	kind    edocuenta.TransactionDirection
	balance int64
}

func inferMissingBalanceTransaction(items []rawTransaction, idx int, running int64) (inferredBalanceTransaction, bool) {
	if idx < 0 || idx >= len(items)-1 {
		return inferredBalanceTransaction{}, false
	}

	current := items[idx]
	next := items[idx+1]
	if current.amountCents <= 0 || next.balanceCents == nil || next.amountCents <= 0 {
		return inferredBalanceTransaction{}, false
	}

	type candidate struct {
		currentKind edocuenta.TransactionDirection
		nextKind    edocuenta.TransactionDirection
	}

	var matches []candidate
	for _, currentKind := range []edocuenta.TransactionDirection{
		edocuenta.TransactionDirectionDebit,
		edocuenta.TransactionDirectionCredit,
	} {
		currentBalance := applyAmount(running, current.amountCents, currentKind)
		for _, nextKind := range []edocuenta.TransactionDirection{
			edocuenta.TransactionDirectionDebit,
			edocuenta.TransactionDirectionCredit,
		} {
			if applyAmount(currentBalance, next.amountCents, nextKind) == *next.balanceCents {
				matches = append(matches, candidate{
					currentKind: currentKind,
					nextKind:    nextKind,
				})
			}
		}
	}

	if len(matches) != 1 {
		return inferredBalanceTransaction{}, false
	}

	match := matches[0]
	if next.kind != "" && next.kind != match.nextKind {
		return inferredBalanceTransaction{}, false
	}

	return inferredBalanceTransaction{
		kind:    match.currentKind,
		balance: applyAmount(running, current.amountCents, match.currentKind),
	}, true
}

func resolveWithDescriptionHints(items []rawTransaction) bool {
	changed := false

	for i := range items {
		if items[i].kind != "" {
			continue
		}

		hint := classifyByDescription(items[i].description)
		if hint == "" {
			continue
		}

		items[i].kind = hint
		changed = true
	}

	return changed
}

func resolveWithSummary(items []rawTransaction, summary *transactionSummary) bool {
	if summary == nil {
		return false
	}

	var (
		resolvedCargoCount int
		resolvedAbonoCount int
		resolvedCargoTotal int64
		resolvedAbonoTotal int64
		unresolvedIndexes  []int
	)

	for i, item := range items {
		switch item.kind {
		case edocuenta.TransactionDirectionDebit:
			resolvedCargoCount++
			resolvedCargoTotal += item.amountCents
		case edocuenta.TransactionDirectionCredit:
			resolvedAbonoCount++
			resolvedAbonoTotal += item.amountCents
		default:
			unresolvedIndexes = append(unresolvedIndexes, i)
		}
	}

	if len(unresolvedIndexes) == 0 {
		return false
	}

	remainingCargoCount := summary.cargoCount - resolvedCargoCount
	remainingAbonoCount := summary.abonoCount - resolvedAbonoCount
	remainingCargoTotal := summary.cargoAmountCents - resolvedCargoTotal
	remainingAbonoTotal := summary.abonoAmountCents - resolvedAbonoTotal

	changed := false

	switch {
	case remainingCargoCount == len(unresolvedIndexes) && remainingAbonoCount == 0:
		for _, idx := range unresolvedIndexes {
			items[idx].kind = edocuenta.TransactionDirectionDebit
			changed = true
		}
	case remainingAbonoCount == len(unresolvedIndexes) && remainingCargoCount == 0:
		for _, idx := range unresolvedIndexes {
			items[idx].kind = edocuenta.TransactionDirectionCredit
			changed = true
		}
	}

	if changed {
		return true
	}

	if len(unresolvedIndexes) == 1 {
		idx := unresolvedIndexes[0]
		switch {
		case remainingCargoCount == 1 && remainingAbonoCount == 0 && items[idx].amountCents == remainingCargoTotal:
			items[idx].kind = edocuenta.TransactionDirectionDebit
			return true
		case remainingAbonoCount == 1 && remainingCargoCount == 0 && items[idx].amountCents == remainingAbonoTotal:
			items[idx].kind = edocuenta.TransactionDirectionCredit
			return true
		}
	}

	return false
}

func classifyByDescription(description string) edocuenta.TransactionDirection {
	upper := strings.ToUpper(description)
	switch {
	case strings.Contains(upper, "SPEI ENVIADO"),
		strings.Contains(upper, "PAGO TARJETA DE CREDITO"),
		strings.Contains(upper, "RETIRO SIN TARJETA"),
		strings.HasPrefix(upper, "SAT"):
		return edocuenta.TransactionDirectionDebit
	case strings.Contains(upper, "SPEI RECIBIDO"),
		strings.Contains(upper, "SPEI RECIBIDOS"),
		strings.Contains(upper, "SPEI DEVUELTO"),
		strings.Contains(upper, "DEPOSITO"),
		strings.Contains(upper, "NOMINA"):
		return edocuenta.TransactionDirectionCredit
	default:
		return ""
	}
}

func applyAmount(balance, amount int64, kind edocuenta.TransactionDirection) int64 {
	if kind == edocuenta.TransactionDirectionCredit {
		return balance + amount
	}
	return balance - amount
}

func legacyMovementKind(value string) edocuenta.TransactionDirection {
	if strings.EqualFold(strings.TrimSpace(value), "ABONO") {
		return edocuenta.TransactionDirectionCredit
	}

	return edocuenta.TransactionDirectionDebit
}

func parseShortDate(value string, periodStart, periodEnd time.Time) (time.Time, error) {
	clean := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(value), " ", ""))
	parts := strings.Split(clean, "/")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid short date %q", value)
	}

	day, err := strconv.Atoi(parts[0])
	if err != nil {
		return time.Time{}, err
	}

	month, ok := normalize.ParseSpanishMonth(parts[1])
	if !ok {
		return time.Time{}, fmt.Errorf("invalid month %q", parts[1])
	}

	for _, year := range []int{periodStart.Year(), periodEnd.Year()} {
		candidate := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		if !candidate.Before(periodStart) && !candidate.After(periodEnd) {
			return candidate, nil
		}
	}

	return time.Date(periodEnd.Year(), month, day, 0, 0, 0, 0, time.UTC), nil
}

func trimNoise(value string) string {
	if idx := strings.IndexRune(value, '�'); idx != -1 {
		value = value[:idx]
	}

	return strings.TrimSpace(value)
}

func looksLikeTransaction(line string) bool {
	if len(line) < 1 {
		return false
	}

	return line[0] >= '0' && line[0] <= '9'
}
