package bbva

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	edocuenta "github.com/DavidSerranoG/go-estado-cuenta-mx"
	"github.com/DavidSerranoG/go-estado-cuenta-mx/internal/normalize"
)

const bbvaCardFullDatePattern = `[0-9]{2}-[A-Za-z]{3,4}-[0-9]{4}`

var (
	cardPeriodPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)periodo\s*:?\s*(` + bbvaCardFullDatePattern + `)\s*al\s*(` + bbvaCardFullDatePattern + `)`),
	}
	cardMaskedAccountPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)n[úu]mero\s*de\s*cuenta:\s*([Xx\*0-9 ]{4,24})`),
		regexp.MustCompile(`(?i)tarjeta\s*titular:\s*([Xx\*0-9 ]{4,24})`),
	}
	cardFullAccountPattern        = regexp.MustCompile(`(?i)n[úu]mero\s*de\s*tarjeta:\s*([0-9 ]{12,24})`)
	cardTransactionLinePattern    = regexp.MustCompile(`([0-9]{2}-[A-Za-z]{3,4}-[0-9]{4})\s*([0-9]{2}-[A-Za-z]{3,4}-[0-9]{4})\s*(.+?)\s*([+-])\s*\$?\s*([0-9,]+\.\d{2})`)
	cardTotalCargosPattern        = regexp.MustCompile(`(?i)TOTAL\s+CARGOS[^0-9+\-$]*([+-]?\s*\$?\s*[0-9,]+\.\d{2})`)
	cardTotalAbonosPattern        = regexp.MustCompile(`(?i)TOTAL\s+ABONOS[^0-9+\-$]*([+-]?\s*\$?\s*[0-9,]+\.\d{2})`)
	cardStatementTitlePattern     = regexp.MustCompile(`(?i)TARJETA\s+.+\s+BBVA`)
	cardStartSectionPattern       = regexp.MustCompile(`(?i)DESGLOSE\s+DE\s+MOVIMIENTOS`)
	cardRegularChargesPattern     = regexp.MustCompile(`(?i)CARGOS,?\s*COMPRAS\s+Y\s+ABONOS\s+REGULARES`)
	cardStopMarkerPattern         = regexp.MustCompile(`(?i)^(ATENCION\s+DE\s+QUEJAS|NOTAS\s+ACLARATORIAS|GLOSARIO\s+DE\s+TERMINOS|DETALLE\s+DE\s+TRANSACCIONES\s+DE\s+BENEFICIOS)`)
	cardPageHeaderPattern         = regexp.MustCompile(`(?i)(n[úu]mero\s+de\s+cuenta:|p[áa]gina\s+\d+\s+de\s+\d+)`)
	cardColumnHeaderPattern       = regexp.MustCompile(`(?i)(fecha\s+de\s+la\s+operaci|fecha\s+de\s+cargo|descripci|monto)`)
	cardMetadataLinePattern       = regexp.MustCompile(`(?i)(iva\s*:?\$|intere[s5]?:\s*\$|comisiones?:\s*\$|capital:?\$|pago\s+excedente|de\s+promocion:?\$)`)
	cardUsefulContinuationPattern = regexp.MustCompile(`(?i)(MXP\s+\$|TIPO\s+DE\s+CAMBIO|TARJETA\s+DIGITAL|AUT:\s*[0-9]+|USD\s+[0-9,]+\.\d{2})`)
)

type retryableCardTextError struct {
	message string
}

func (e retryableCardTextError) Error() string {
	return e.message
}

func (retryableCardTextError) RetryableTextInsufficiency() {}

func newRetryableCardTextError(message string) error {
	return retryableCardTextError{message: message}
}

func looksLikeCardStatement(text string) bool {
	upper := strings.ToUpper(text)
	return cardStatementTitlePattern.MatchString(text) ||
		(strings.Contains(upper, "TU PAGO REQUERIDO ESTE PERIODO") && matchesAnyRegexp(text, cardPeriodPatterns...)) ||
		(strings.Contains(upper, "DESGLOSE DE MOVIMIENTOS") && strings.Contains(upper, "TOTAL CARGOS") && strings.Contains(upper, "TOTAL ABONOS"))
}

func cardDetectionScore(text string) int {
	if !looksLikeCardStatement(text) {
		return 0
	}

	upper := strings.ToUpper(text)
	score := 2

	if cardStatementTitlePattern.MatchString(text) {
		score += 2
	}
	if matchesAnyRegexp(text, cardPeriodPatterns...) {
		score += 2
	}
	if matchesAnyRegexp(text, cardMaskedAccountPatterns...) || cardFullAccountPattern.MatchString(text) {
		score += 2
	}
	if strings.Contains(upper, "TU PAGO REQUERIDO ESTE PERIODO") {
		score += 2
	}
	if strings.Contains(upper, "PAGO PARA NO GENERAR INTERESES") {
		score++
	}
	if cardStartSectionPattern.MatchString(text) {
		score += 2
	}
	if cardRegularChargesPattern.MatchString(text) {
		score++
	}
	if strings.Contains(upper, "TOTAL CARGOS") {
		score++
	}
	if strings.Contains(upper, "TOTAL ABONOS") {
		score++
	}
	if cardTransactionLinePattern.MatchString(text) {
		score += 2
	}

	return score
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

	transactions, warnings, err := parseCardTransactions(text)
	if err != nil {
		return edocuenta.ParseResult{}, err
	}

	return edocuenta.ParseResult{
		Statement: edocuenta.Statement{
			Bank:          edocuenta.BankBBVA,
			AccountNumber: account,
			Currency:      edocuenta.CurrencyMXN,
			PeriodStart:   periodStart,
			PeriodEnd:     periodEnd,
			Transactions:  transactions,
		},
		Warnings: warnings,
	}, nil
}

func parseCardAccount(text string) (string, error) {
	for _, pattern := range cardMaskedAccountPatterns {
		match := pattern.FindStringSubmatch(text)
		if len(match) != 2 {
			continue
		}

		identifier := normalizeMaskedIdentifier(match[1])
		if identifier != "" {
			return identifier, nil
		}
	}

	match := cardFullAccountPattern.FindStringSubmatch(text)
	if len(match) != 2 {
		return "", fmt.Errorf("bbva: credit card account number not found")
	}

	account := normalize.DigitsOnly(match[1])
	if len(account) < 12 {
		return "", fmt.Errorf("bbva: credit card account number not found")
	}

	return account, nil
}

func normalizeMaskedIdentifier(value string) string {
	clean := strings.ToUpper(strings.TrimSpace(value))
	if clean == "" {
		return ""
	}

	var b strings.Builder
	for _, r := range clean {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == 'X' || r == '*':
			b.WriteRune('X')
		}
	}

	result := b.String()
	if result == "" {
		return ""
	}
	if strings.ContainsRune(result, 'X') && len(result) >= 4 {
		return result
	}
	return ""
}

func parseCardPeriod(text string) (time.Time, time.Time, error) {
	for _, pattern := range cardPeriodPatterns {
		match := pattern.FindStringSubmatch(text)
		if len(match) != 3 {
			continue
		}

		start, err := normalize.ParseDateDDMonYYYYSpanish(match[1])
		if err != nil {
			return time.Time{}, time.Time{}, err
		}

		end, err := normalize.ParseDateDDMonYYYYSpanish(match[2])
		if err != nil {
			return time.Time{}, time.Time{}, err
		}

		return start, end, nil
	}

	return time.Time{}, time.Time{}, fmt.Errorf("bbva: credit card period not found")
}

func parseCardTransactions(text string) ([]edocuenta.Transaction, []string, error) {
	section, ok := extractCardTransactionSection(text)
	if !ok {
		return nil, nil, newRetryableCardTextError("bbva: credit card text incomplete: movement section not found")
	}

	totalCargos, totalAbonos, ok := parseCardTotals(section)
	if !ok {
		return nil, nil, newRetryableCardTextError("bbva: credit card text incomplete: movement totals not found")
	}

	var (
		transactions []edocuenta.Transaction
		warnings     []string
		lastIndex    = -1
	)

	for _, rawLine := range strings.Split(section, "\n") {
		line := normalize.CollapseWhitespace(trimNoise(rawLine))
		if line == "" {
			continue
		}
		if cardStopMarkerPattern.MatchString(line) {
			break
		}
		if shouldIgnoreCardLine(line) {
			continue
		}

		matches := cardTransactionLinePattern.FindAllStringSubmatch(line, -1)
		if len(matches) > 0 {
			for _, match := range matches {
				transaction, err := buildCardTransaction(match)
				if err != nil {
					warnings = append(warnings, err.Error()+": "+line)
					continue
				}

				transactions = append(transactions, transaction)
				lastIndex = len(transactions) - 1
			}
			continue
		}

		if lastIndex >= 0 && isCardContinuationLine(line) {
			transactions[lastIndex].Description = normalize.CollapseWhitespace(transactions[lastIndex].Description + " " + line)
		}
	}

	if len(transactions) == 0 {
		return nil, warnings, newRetryableCardTextError("bbva: credit card text incomplete: no transactions found")
	}

	if err := validateCardTotals(transactions, totalCargos, totalAbonos); err != nil {
		return nil, warnings, err
	}

	return transactions, warnings, nil
}

func extractCardTransactionSection(text string) (string, bool) {
	index := cardStartSectionPattern.FindStringIndex(text)
	if index == nil {
		return "", false
	}

	return text[index[0]:], true
}

func parseCardTotals(text string) (int64, int64, bool) {
	cargosMatch := cardTotalCargosPattern.FindStringSubmatch(text)
	abonosMatch := cardTotalAbonosPattern.FindStringSubmatch(text)
	if len(cargosMatch) != 2 || len(abonosMatch) != 2 {
		return 0, 0, false
	}

	cargos, err := parseAbsoluteMoneyToken(cargosMatch[1])
	if err != nil {
		return 0, 0, false
	}

	abonos, err := parseAbsoluteMoneyToken(abonosMatch[1])
	if err != nil {
		return 0, 0, false
	}

	return cargos, abonos, true
}

func shouldIgnoreCardLine(line string) bool {
	upper := strings.ToUpper(line)

	switch {
	case cardStartSectionPattern.MatchString(line),
		cardRegularChargesPattern.MatchString(line),
		cardPageHeaderPattern.MatchString(line),
		cardColumnHeaderPattern.MatchString(line),
		cardMetadataLinePattern.MatchString(line):
		return true
	case strings.Contains(upper, "TOTAL CARGOS"),
		strings.Contains(upper, "TOTAL ABONOS"),
		strings.Contains(upper, "DESGLOSE DE MOVIMIENTOS"),
		strings.HasPrefix(upper, "NOTAS:"),
		strings.Contains(upper, "MENSAJES ADICIONALES"),
		strings.Contains(upper, "PROGRAMA DE BENEFICIOS"),
		strings.Contains(upper, "SALDO SOBRE EL QUE SE CALCULARON"),
		strings.Contains(upper, "DISTRIBUCION DE TU ULTIMO PAGO"):
		return true
	default:
		return false
	}
}

func isCardContinuationLine(line string) bool {
	if cardTransactionLinePattern.MatchString(line) || shouldIgnoreCardLine(line) {
		return false
	}

	upper := strings.ToUpper(line)
	switch {
	case strings.Contains(upper, "TOTAL "),
		strings.Contains(upper, "ATENCION DE QUEJAS"),
		strings.Contains(upper, "NOTAS ACLARATORIAS"),
		strings.Contains(upper, "PAGINA "),
		strings.Contains(upper, "NUMERO DE CUENTA"):
		return false
	}

	return cardUsefulContinuationPattern.MatchString(line)
}

func buildCardTransaction(match []string) (edocuenta.Transaction, error) {
	if len(match) != 6 {
		return edocuenta.Transaction{}, fmt.Errorf("bbva: invalid credit card transaction")
	}

	postedAt, err := normalize.ParseDateDDMonYYYYSpanish(match[2])
	if err != nil {
		return edocuenta.Transaction{}, fmt.Errorf("bbva: invalid credit card posting date")
	}

	amountCents, err := parseAbsoluteMoneyToken(match[5])
	if err != nil {
		return edocuenta.Transaction{}, fmt.Errorf("bbva: invalid credit card amount")
	}

	description := normalize.CollapseWhitespace(strings.TrimSpace(match[3]))

	return edocuenta.Transaction{
		PostedAt:    postedAt,
		Description: description,
		Kind:        cardMovementKind(match[4]),
		AmountCents: amountCents,
	}, nil
}

func cardMovementKind(sign string) edocuenta.TransactionKind {
	if strings.TrimSpace(sign) == "-" {
		return edocuenta.TransactionKindCredit
	}
	return edocuenta.TransactionKindDebit
}

func parseAbsoluteMoneyToken(value string) (int64, error) {
	cents, err := parseMoneyToken(value)
	if err != nil {
		return 0, err
	}
	if cents < 0 {
		return -cents, nil
	}
	return cents, nil
}

func parseMoneyToken(value string) (int64, error) {
	clean := strings.TrimSpace(value)
	clean = strings.ReplaceAll(clean, "$", "")
	clean = strings.ReplaceAll(clean, " ", "")
	clean = strings.TrimLeft(clean, "|)")
	return normalize.ParseOCRMoneyToCents(clean)
}

func validateCardTotals(transactions []edocuenta.Transaction, totalCargos, totalAbonos int64) error {
	var cargos, abonos int64

	for _, transaction := range transactions {
		switch transaction.Kind {
		case edocuenta.TransactionKindDebit:
			cargos += transaction.AmountCents
		case edocuenta.TransactionKindCredit:
			abonos += transaction.AmountCents
		}
	}

	if cargos != totalCargos || abonos != totalAbonos {
		return newRetryableCardTextError(
			fmt.Sprintf(
				"bbva: credit card text incomplete: parsed totals cargos=%d abonos=%d want cargos=%d abonos=%d",
				cargos,
				abonos,
				totalCargos,
				totalAbonos,
			),
		)
	}

	return nil
}
