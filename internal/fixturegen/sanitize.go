package fixturegen

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	edocuenta "github.com/DavidSerranoG/go-estado-cuenta-mx"
	"github.com/DavidSerranoG/go-estado-cuenta-mx/internal/normalize"
)

var (
	reSlashDate      = regexp.MustCompile(`\b\d{2}/\d{2}/\d{4}\b`)
	reMonDate        = regexp.MustCompile(`\b\d{2}-[A-Za-z0-9]{3,5}-\d{4}\b`)
	reReferenceToken = regexp.MustCompile(`\b[A-Z0-9]{8,}\b`)

	reBBVACardLine  = regexp.MustCompile(`^(\d{2}-[A-Za-z0-9]{3,5}-\d{4})\s+(\d{2}-[A-Za-z0-9]{3,5}-\d{4})\s+(.+?)\s+([+-])\s+\$?\s*([0-9,]+\.\d{2})$`)
	reBBVATotalLine = regexp.MustCompile(`(?i)^(TOTAL\s+(?:CARGOS|ABONOS))\s+([+-]?\$?\s*[0-9,]+\.\d{2})$`)
	reBBVAOpenBal   = regexp.MustCompile(`(?i)^(saldo anterior)\s+([0-9,]+\.\d{2})$`)
	reBBVASummary   = regexp.MustCompile(`(?i)(total importe cargos\s*)([0-9,]+\.\d{2})(\s*total movimientos cargos\s*)([0-9]+)(\s*total importe abonos\s*)([0-9,]+\.\d{2})(\s*total movimientos abonos\s*)([0-9]+)`)
	reBBVARealTx    = regexp.MustCompile(`^(\d{2}\s*/\s*[A-Z]{3})\s*(\d{2}\s*/\s*[A-Z]{3})\s*(.+?)([0-9,]+\.\d{2})([0-9,]+\.\d{2})([0-9,]+\.\d{2})\s*(.*)$`)
	reBBVALegacyTx  = regexp.MustCompile(`^(\d{2}/\d{2}/\d{4})\s+(.+?)\s+(ABONO|CARGO)\s+([0-9,]+\.\d{2})\s+([0-9,]+\.\d{2})$`)

	reHSBCCardLine     = regexp.MustCompile(`^(\d{2}-[A-Za-z0-9]{3,5}-\d{4})\s+(\d{2}-[A-Za-z0-9]{3,5}-\d{4})\s+(.+?)\s+([+-])\s+\$?\s*([0-9,]+\.\d{2})$`)
	reHSBCFlexibleLine = regexp.MustCompile(`^(\d{2})(.+)$`)
	reHSBCFlexibleAmt  = regexp.MustCompile(`^\$\s*([0-9,]+\.\d{2})\s+\$\s*([0-9,]+\.\d{2})$`)
	reHSBCInitialBal   = regexp.MustCompile(`(?i)^(saldo inicial del\s*periodo)\s+\$?\s*([0-9,]+\.\d{2})$`)
	rePeriodDDMon      = regexp.MustCompile(`(?i)(\d{2}-[A-Za-z0-9]{3,5}-\d{4})\s+al\s+(\d{2}-[A-Za-z0-9]{3,5}-\d{4})`)
	rePeriodCompact    = regexp.MustCompile(`(?i)(\d{8})\s+al\s+(\d{8})`)
)

type sanitizeContext struct {
	hint              Hint
	original          edocuenta.ParseResult
	dummy             edocuenta.Statement
	dateShift         int
	replacements      map[string]*Replacement
	accountVariants   map[string]string
	referenceVariants map[string]string
	nextTx            int
	nextBalance       int
	debitTotal        int64
	creditTotal       int64
	openingBalance    *int64
	overrides         Overrides
}

func newSanitizeContext(hint Hint, original edocuenta.ParseResult, overrides Overrides) *sanitizeContext {
	ctx := &sanitizeContext{
		hint:              normalizeHint(hint, original),
		original:          original,
		dateShift:         47 + (len(original.Statement.Transactions) % 89),
		replacements:      map[string]*Replacement{},
		accountVariants:   map[string]string{},
		referenceVariants: map[string]string{},
		overrides:         overrides,
	}

	ctx.dummy = buildDummyStatement(ctx)
	ctx.debitTotal, ctx.creditTotal = statementTotals(ctx.dummy)
	ctx.openingBalance = deriveOpeningBalance(ctx.dummy)
	return ctx
}

func normalizeHint(hint Hint, original edocuenta.ParseResult) Hint {
	result := hint
	if result.Bank == "" {
		result.Bank = strings.ToLower(string(original.Statement.Bank))
	}
	if result.Layout == "" {
		result.Layout = inferLayout(original)
	}
	result.File = filepath.Base(result.File)
	return result
}

func inferLayout(original edocuenta.ParseResult) string {
	text := strings.ToUpper(original.ExtractedText)
	switch {
	case strings.Contains(text, "CUENTA FLEXIBLE"):
		return "flexible"
	case strings.Contains(text, "TARJETA") || strings.Contains(text, "TU PAGO REQUERIDO ESTE PERIODO"):
		return "card"
	default:
		return "account"
	}
}

func buildDummyStatement(ctx *sanitizeContext) edocuenta.Statement {
	statement := ctx.original.Statement
	statement.AccountNumber = sanitizeAccount(statement.AccountNumber, ctx)
	statement.PeriodStart = statement.PeriodStart.AddDate(0, 0, ctx.dateShift)
	statement.PeriodEnd = statement.PeriodEnd.AddDate(0, 0, ctx.dateShift)

	transactions := make([]edocuenta.Transaction, 0, len(statement.Transactions))
	running := int64(1000000 + len(statement.Transactions)*75000)
	if hasRunningBalances(statement.Transactions) {
		running = deriveSeedOpening(statement.Transactions)
	}

	for i, tx := range statement.Transactions {
		dummy := tx
		dummy.PostedAt = tx.PostedAt.AddDate(0, 0, ctx.dateShift)
		dummy.Reference = sanitizeReference(tx.Reference, i, ctx)
		dummy.Description = sanitizeDescription(tx.Description, i, ctx)
		dummy.AmountCents = dummyAmount(tx.AmountCents, i)
		if tx.BalanceCents != nil {
			if tx.Kind == edocuenta.TransactionKindCredit {
				running += dummy.AmountCents
			} else {
				running -= dummy.AmountCents
			}
			balance := running
			dummy.BalanceCents = &balance
		}
		transactions = append(transactions, dummy)
	}

	statement.Transactions = transactions
	return statement
}

func hasRunningBalances(transactions []edocuenta.Transaction) bool {
	for _, tx := range transactions {
		if tx.BalanceCents != nil {
			return true
		}
	}
	return false
}

func deriveSeedOpening(transactions []edocuenta.Transaction) int64 {
	var total int64 = 2500000
	for i, tx := range transactions {
		total += int64((i + 1) * 6500)
		total += tx.AmountCents / 10
	}
	if total < 500000 {
		total = 500000
	}
	return total
}

func dummyAmount(original int64, idx int) int64 {
	wholeDigits := len(strconvAbs(original / 100))
	if wholeDigits < 2 {
		wholeDigits = 2
	}

	base := strings.Repeat("7", wholeDigits-1) + fmt.Sprintf("%d", (idx%7)+1)
	whole := int64(0)
	for _, r := range base {
		whole = (whole * 10) + int64(r-'0')
	}

	cents := int64((idx*17)%90 + 10)
	return whole*100 + cents
}

func statementTotals(statement edocuenta.Statement) (int64, int64) {
	var debit, credit int64
	for _, tx := range statement.Transactions {
		if tx.Kind == edocuenta.TransactionKindCredit {
			credit += tx.AmountCents
		} else {
			debit += tx.AmountCents
		}
	}
	return debit, credit
}

func deriveOpeningBalance(statement edocuenta.Statement) *int64 {
	if len(statement.Transactions) == 0 {
		return nil
	}
	first := statement.Transactions[0]
	if first.BalanceCents == nil {
		return nil
	}

	value := *first.BalanceCents
	if first.Kind == edocuenta.TransactionKindCredit {
		value -= first.AmountCents
	} else {
		value += first.AmountCents
	}
	return &value
}

func sanitizeAccount(value string, ctx *sanitizeContext) string {
	if value == "" {
		return ""
	}

	masked := strings.ContainsAny(value, "Xx*")
	var out strings.Builder
	digitCount := 0
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
			digitCount++
			out.WriteByte(byte('0' + ((digitCount + 3) % 10)))
		case r == 'X' || r == 'x' || r == '*':
			if masked {
				out.WriteByte('X')
			} else {
				out.WriteRune(r)
			}
		default:
			out.WriteRune(r)
		}
	}

	return out.String()
}

func sanitizeReference(value string, idx int, ctx *sanitizeContext) string {
	if value == "" {
		return ""
	}
	if dummy, ok := ctx.referenceVariants[value]; ok {
		return dummy
	}

	var out strings.Builder
	counter := idx + 3
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
			out.WriteByte(byte('0' + (counter % 10)))
			counter++
		case r >= 'A' && r <= 'Z':
			out.WriteByte(byte('A' + (counter % 26)))
			counter++
		case r >= 'a' && r <= 'z':
			out.WriteByte(byte('a' + (counter % 26)))
			counter++
		default:
			out.WriteRune(r)
		}
	}

	result := out.String()
	ctx.referenceVariants[value] = result
	return result
}

func sanitizeDescription(value string, idx int, ctx *sanitizeContext) string {
	if value == "" {
		return ""
	}

	safe := map[string]struct{}{
		"SPEI": {}, "DEPOSITO": {}, "NOMINA": {}, "SAT": {},
		"PAGO": {}, "HSBC": {}, "BBVA": {}, "CUENTA": {}, "FLEXIBLE": {},
		"MONEDA": {}, "EXTRANJERA": {}, "USD": {}, "TC": {}, "TARJETA": {},
	}

	parts := strings.Fields(value)
	for i, part := range parts {
		upper := strings.ToUpper(strings.Trim(part, ".,;:()"))
		if upper == "" {
			continue
		}
		if _, ok := safe[upper]; ok {
			continue
		}
		if reReferenceToken.MatchString(upper) {
			parts[i] = sanitizeReference(part, idx+i, ctx)
			continue
		}
		if strings.IndexFunc(upper, func(r rune) bool { return r >= 'A' && r <= 'Z' }) == -1 {
			continue
		}
		parts[i] = pseudoWord(part, idx+i)
	}

	return strings.Join(parts, " ")
}

func pseudoWord(value string, seed int) string {
	letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lower := "abcdefghijklmnopqrstuvwxyz"
	var out strings.Builder
	pos := seed + 5
	for _, r := range value {
		switch {
		case r >= 'A' && r <= 'Z':
			out.WriteByte(letters[pos%len(letters)])
			pos++
		case r >= 'a' && r <= 'z':
			out.WriteByte(lower[pos%len(lower)])
			pos++
		case r >= '0' && r <= '9':
			out.WriteByte(byte('0' + (pos % 10)))
			pos++
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}

func (ctx *sanitizeContext) sanitizeTextDocument(document TextDocument, branding BrandingMode) TextDocument {
	result := TextDocument{Tool: document.Tool, Pages: make([]TextPage, 0, len(document.Pages))}
	for _, page := range document.Pages {
		sanitizedPage := TextPage{
			Number: page.Number,
			Width:  page.Width,
			Height: page.Height,
			Lines:  make([]TextLine, 0, len(page.Lines)),
		}
		for _, line := range page.Lines {
			sanitizedPage.Lines = append(sanitizedPage.Lines, TextLine{
				Text: ctx.sanitizeLine(line.Text, branding),
				XMin: line.XMin,
				YMin: line.YMin,
				XMax: line.XMax,
				YMax: line.YMax,
			})
		}
		result.Pages = append(result.Pages, sanitizedPage)
	}
	return result
}

func (ctx *sanitizeContext) sanitizeLine(line string, branding BrandingMode) string {
	line = normalize.NormalizeOCRLine(line)
	if line == "" {
		return ""
	}

	for _, replacement := range ctx.overrides.LineReplacements {
		if replacement.Match == "" {
			continue
		}
		line = strings.ReplaceAll(line, replacement.Match, replacement.Replace)
	}

	switch {
	case ctx.hint.Bank == "bbva" && ctx.hint.Layout == "card":
		line = ctx.sanitizeBBVACardLine(line)
	case ctx.hint.Bank == "bbva":
		line = ctx.sanitizeBBVAAccountLine(line)
	case ctx.hint.Bank == "hsbc" && ctx.hint.Layout == "flexible":
		line = ctx.sanitizeHSBCFlexibleLine(line)
	case ctx.hint.Bank == "hsbc":
		line = ctx.sanitizeHSBCCardLine(line)
	}

	line = ctx.replaceStructuredValues(line, branding)
	line = ctx.replaceDates(line)
	if branding == BrandingNeutral {
		line = neutralizeBrand(line)
	}

	return line
}

func (ctx *sanitizeContext) sanitizeBBVACardLine(line string) string {
	if match := reBBVACardLine.FindStringSubmatch(line); len(match) == 6 && ctx.nextTx < len(ctx.dummy.Transactions) {
		tx := ctx.dummy.Transactions[ctx.nextTx]
		ctx.nextTx++
		return fmt.Sprintf("%s %s %s %s $%s",
			formatDateLike(match[1], tx.PostedAt.AddDate(0, 0, -1)),
			formatDateLike(match[2], tx.PostedAt),
			tx.Description,
			cardSign(tx.Kind),
			formatMoney(tx.AmountCents),
		)
	}
	if match := reBBVATotalLine.FindStringSubmatch(line); len(match) == 3 {
		if strings.Contains(strings.ToUpper(match[1]), "CARGOS") {
			return match[1] + " $" + formatMoney(ctx.debitTotal)
		}
		return match[1] + " -$" + formatMoney(ctx.creditTotal)
	}
	return line
}

func (ctx *sanitizeContext) sanitizeBBVAAccountLine(line string) string {
	if match := reBBVALegacyTx.FindStringSubmatch(line); len(match) == 6 && ctx.nextTx < len(ctx.dummy.Transactions) {
		tx := ctx.dummy.Transactions[ctx.nextTx]
		ctx.nextTx++
		balance := int64(0)
		if tx.BalanceCents != nil {
			balance = *tx.BalanceCents
		}
		return fmt.Sprintf("%s %s %s %s %s",
			formatDateLike(match[1], tx.PostedAt),
			tx.Description,
			legacyKind(tx.Kind),
			formatMoney(tx.AmountCents),
			formatMoney(balance),
		)
	}
	if match := reBBVARealTx.FindStringSubmatch(line); len(match) == 7 && ctx.nextTx < len(ctx.dummy.Transactions) {
		tx := ctx.dummy.Transactions[ctx.nextTx]
		ctx.nextTx++
		operationDate := tx.PostedAt.AddDate(0, 0, -1)
		balance := int64(0)
		if tx.BalanceCents != nil {
			balance = *tx.BalanceCents
		}
		return fmt.Sprintf("%s %s %s%s%s %s",
			formatShortDate(operationDate),
			formatShortDate(tx.PostedAt),
			tx.Description,
			formatMoney(tx.AmountCents),
			formatMoney(balance),
			sanitizeReference(strings.TrimSpace(match[6]), ctx.nextTx, ctx),
		)
	}
	if match := reBBVAOpenBal.FindStringSubmatch(line); len(match) == 3 && ctx.openingBalance != nil {
		return match[1] + " " + formatMoney(*ctx.openingBalance)
	}
	if reBBVASummary.MatchString(line) {
		return reBBVASummary.ReplaceAllString(line,
			fmt.Sprintf("${1}%s${3}${4}${5}%s${7}${8}",
				formatMoney(ctx.debitTotal),
				formatMoney(ctx.creditTotal),
			),
		)
	}
	return line
}

func (ctx *sanitizeContext) sanitizeHSBCCardLine(line string) string {
	if match := reHSBCCardLine.FindStringSubmatch(line); len(match) == 6 && ctx.nextTx < len(ctx.dummy.Transactions) {
		tx := ctx.dummy.Transactions[ctx.nextTx]
		ctx.nextTx++
		return fmt.Sprintf("%s %s %s %s $%s",
			formatDateLike(match[1], tx.PostedAt.AddDate(0, 0, -1)),
			formatDateLike(match[2], tx.PostedAt),
			tx.Description,
			hsbcSign(tx.Kind),
			formatMoney(tx.AmountCents),
		)
	}
	return line
}

func (ctx *sanitizeContext) sanitizeHSBCFlexibleLine(line string) string {
	if match := reHSBCInitialBal.FindStringSubmatch(line); len(match) == 3 && ctx.openingBalance != nil {
		return match[1] + " $" + formatMoney(*ctx.openingBalance)
	}
	if match := reHSBCFlexibleAmt.FindStringSubmatch(line); len(match) == 3 && ctx.nextBalance < len(ctx.dummy.Transactions) {
		tx := ctx.dummy.Transactions[ctx.nextBalance]
		ctx.nextBalance++
		balance := int64(0)
		if tx.BalanceCents != nil {
			balance = *tx.BalanceCents
		}

		if tx.Kind == edocuenta.TransactionKindCredit {
			return fmt.Sprintf("$ %s $ %s", formatMoney(tx.AmountCents), formatMoney(balance))
		}
		return fmt.Sprintf("$ %s $ %s", formatMoney(tx.AmountCents), formatMoney(balance))
	}
	if match := reHSBCFlexibleLine.FindStringSubmatch(line); len(match) == 3 && ctx.nextTx < len(ctx.dummy.Transactions) {
		tx := ctx.dummy.Transactions[ctx.nextTx]
		ctx.nextTx++
		desc := tx.Description
		if tx.Reference != "" {
			desc = desc + " " + tx.Reference
		}
		return fmt.Sprintf("%02d%s", tx.PostedAt.Day(), padPreservePrefix(match[2], desc))
	}
	return line
}

func (ctx *sanitizeContext) replaceStructuredValues(line string, branding BrandingMode) string {
	variants := ctx.accountVariants
	origAccount := ctx.original.Statement.AccountNumber
	if origAccount != "" {
		dummyAccount := ctx.dummy.AccountNumber
		for _, variant := range accountRepresentations(origAccount) {
			variants[variant] = accountVariant(origAccount, dummyAccount, variant)
		}
	}

	for original, replacement := range variants {
		line = replaceExact(line, original, replacement)
		ctx.trackReplacement("account", original, replacement, line)
	}

	for _, tx := range ctx.original.Statement.Transactions {
		if tx.Reference == "" {
			continue
		}
		dummy := sanitizeReference(tx.Reference, 0, ctx)
		line = replaceExact(line, tx.Reference, dummy)
		ctx.trackReplacement("reference", tx.Reference, dummy, line)
	}

	return line
}

func replaceExact(line, original, replacement string) string {
	if original == "" || original == replacement {
		return line
	}
	return strings.ReplaceAll(line, original, replacement)
}

func (ctx *sanitizeContext) replaceDates(line string) string {
	line = rePeriodDDMon.ReplaceAllStringFunc(line, func(value string) string {
		match := rePeriodDDMon.FindStringSubmatch(value)
		if len(match) != 3 {
			return value
		}
		start, err := normalize.ParseDateDDMonYYYYSpanish(match[1])
		if err != nil {
			return value
		}
		end, err := normalize.ParseDateDDMonYYYYSpanish(match[2])
		if err != nil {
			return value
		}
		return formatDateLike(match[1], start.AddDate(0, 0, ctx.dateShift)) + " al " + formatDateLike(match[2], end.AddDate(0, 0, ctx.dateShift))
	})

	line = rePeriodCompact.ReplaceAllStringFunc(line, func(value string) string {
		match := rePeriodCompact.FindStringSubmatch(value)
		if len(match) != 3 {
			return value
		}
		start, err := time.Parse("02012006", match[1])
		if err != nil {
			return value
		}
		end, err := time.Parse("02012006", match[2])
		if err != nil {
			return value
		}
		return start.AddDate(0, 0, ctx.dateShift).Format("02012006") + " al " + end.AddDate(0, 0, ctx.dateShift).Format("02012006")
	})

	line = reSlashDate.ReplaceAllStringFunc(line, func(value string) string {
		date, err := normalize.ParseDateDDMMYYYY(value)
		if err != nil {
			return value
		}
		replacement := date.AddDate(0, 0, ctx.dateShift).Format("02/01/2006")
		ctx.trackReplacement("date", value, replacement, line)
		return replacement
	})

	line = reMonDate.ReplaceAllStringFunc(line, func(value string) string {
		date, err := normalize.ParseDateDDMonYYYYSpanish(value)
		if err != nil {
			return value
		}
		replacement := formatDateLike(value, date.AddDate(0, 0, ctx.dateShift))
		ctx.trackReplacement("date", value, replacement, line)
		return replacement
	})

	return line
}

func accountRepresentations(value string) []string {
	repr := []string{value}
	digits := normalize.DigitsOnly(value)
	if digits != "" {
		repr = append(repr, digits)
		repr = append(repr, spacedDigits(digits, 4))
	}
	sort.SliceStable(repr, func(i, j int) bool {
		return len(repr[i]) > len(repr[j])
	})

	seen := map[string]struct{}{}
	unique := make([]string, 0, len(repr))
	for _, item := range repr {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		unique = append(unique, item)
	}
	return unique
}

func accountVariant(original, dummy, variant string) string {
	if strings.ContainsAny(variant, "Xx*") {
		return sanitizeAccount(variant, nil)
	}
	dummyDigits := normalize.DigitsOnly(dummy)
	if strings.Contains(variant, " ") {
		return spacedDigits(dummyDigits, 4)
	}
	if normalize.DigitsOnly(variant) == variant {
		return dummyDigits
	}
	return dummy
}

func spacedDigits(value string, chunk int) string {
	if value == "" || chunk <= 0 {
		return value
	}

	parts := make([]string, 0, (len(value)+chunk-1)/chunk)
	for len(value) > chunk {
		parts = append(parts, value[:chunk])
		value = value[chunk:]
	}
	parts = append(parts, value)
	return strings.Join(parts, " ")
}

func formatMoney(cents int64) string {
	negative := cents < 0
	if negative {
		cents = -cents
	}

	whole := cents / 100
	fraction := cents % 100
	text := insertThousands(fmt.Sprintf("%d", whole)) + fmt.Sprintf(".%02d", fraction)
	if negative {
		return "-" + text
	}
	return text
}

func insertThousands(value string) string {
	if len(value) <= 3 {
		return value
	}

	var parts []string
	for len(value) > 3 {
		parts = append(parts, value[len(value)-3:])
		value = value[:len(value)-3]
	}
	parts = append(parts, value)
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, ",")
}

func formatDateLike(template string, date time.Time) string {
	switch {
	case strings.Contains(template, "/"):
		return date.Format("02/01/2006")
	case strings.Contains(strings.ToLower(template), "-"):
		month := monthToken(date.Month(), template)
		return fmt.Sprintf("%02d-%s-%04d", date.Day(), month, date.Year())
	default:
		return date.Format("02/01/2006")
	}
}

func monthToken(month time.Month, template string) string {
	base := map[time.Month]string{
		time.January:   "Jan",
		time.February:  "Feb",
		time.March:     "Mar",
		time.April:     "Abr",
		time.May:       "May",
		time.June:      "Jun",
		time.July:      "Jul",
		time.August:    "Ago",
		time.September: "Sep",
		time.October:   "Oct",
		time.November:  "Nov",
		time.December:  "Dic",
	}[month]

	if strings.ToUpper(template) == template {
		return strings.ToUpper(base)
	}
	if strings.ToLower(template) == template {
		return strings.ToLower(base)
	}
	return base
}

func formatShortDate(date time.Time) string {
	return fmt.Sprintf("%02d/%s", date.Day(), strings.ToUpper(monthToken(date.Month(), "SEP")))
}

func cardSign(kind edocuenta.TransactionKind) string {
	if kind == edocuenta.TransactionKindCredit {
		return "-"
	}
	return "+"
}

func hsbcSign(kind edocuenta.TransactionKind) string {
	if kind == edocuenta.TransactionKindCredit {
		return "-"
	}
	return "+"
}

func legacyKind(kind edocuenta.TransactionKind) string {
	if kind == edocuenta.TransactionKindCredit {
		return "ABONO"
	}
	return "CARGO"
}

func padPreservePrefix(template, replacement string) string {
	if replacement == "" {
		return template
	}
	return " " + replacement
}

func neutralizeBrand(line string) string {
	replacer := strings.NewReplacer(
		"BBVA", "BANCO",
		"BANCOMER", "BANCO",
		"HSBC", "BANCO",
		"MEXICO", "DEMO",
	)
	return replacer.Replace(line)
}

func (ctx *sanitizeContext) trackReplacement(kind, original, replacement, line string) {
	if original == "" || original == replacement || !strings.Contains(line, replacement) {
		return
	}

	key := kind + ":" + original
	item := ctx.replacements[key]
	if item == nil {
		item = &Replacement{
			Type:           kind,
			OriginalHash:   hashValue(original),
			Replacement:    replacement,
			ContextPreview: previewLine(line, replacement),
		}
		ctx.replacements[key] = item
	}
	item.Occurrences++
}

func hashValue(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:8])
}

func previewLine(line, replacement string) string {
	if replacement == "" || !strings.Contains(line, replacement) {
		return ""
	}
	line = strings.ReplaceAll(line, replacement, "["+replacement+"]")
	if len(line) > 120 {
		return line[:120]
	}
	return line
}

func replacementList(items map[string]*Replacement) []Replacement {
	out := make([]Replacement, 0, len(items))
	for _, item := range items {
		out = append(out, *item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Type == out[j].Type {
			return out[i].OriginalHash < out[j].OriginalHash
		}
		return out[i].Type < out[j].Type
	})
	return out
}

func replacementCounts(items map[string]*Replacement) map[string]int {
	counts := map[string]int{}
	for _, item := range items {
		counts[item.Type] += item.Occurrences
	}
	return counts
}

func strconvAbs(value int64) string {
	if value < 0 {
		value = -value
	}
	return fmt.Sprintf("%d", value)
}
