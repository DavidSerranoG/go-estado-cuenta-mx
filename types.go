package edocuenta

import "time"

// Bank identifies a supported bank.
type Bank string

const (
	BankBBVA Bank = "bbva"
	BankHSBC Bank = "hsbc"
)

// Currency identifies the statement currency.
type Currency string

const (
	CurrencyMXN Currency = "MXN"
	CurrencyUSD Currency = "USD"
)

// AccountClass classifies the statement account using accounting terminology.
type AccountClass string

const (
	AccountClassUnknown   AccountClass = ""
	AccountClassAsset     AccountClass = "asset"
	AccountClassLiability AccountClass = "liability"
)

// TransactionDirection classifies the normalized transaction direction.
type TransactionDirection string

const (
	TransactionDirectionDebit  TransactionDirection = "debit"
	TransactionDirectionCredit TransactionDirection = "credit"
)

// Statement contains normalized domain data extracted from a bank statement.
//
// It intentionally excludes extraction and parsing diagnostics so callers can
// treat it as a clean domain model.
type Statement struct {
	Bank          Bank
	AccountNumber string
	Currency      Currency
	PeriodStart   time.Time
	PeriodEnd     time.Time
	AccountClass  AccountClass      `json:",omitempty"`
	Summary       *StatementSummary `json:",omitempty"`
	Transactions  []Transaction
}

// StatementSummary contains explicit statement-level balances, totals, and
// payment metadata when the source document exposes them clearly.
type StatementSummary struct {
	OpeningBalanceCents         *int64     `json:",omitempty"`
	ClosingBalanceCents         *int64     `json:",omitempty"`
	AverageBalanceCents         *int64     `json:",omitempty"`
	TotalDebitsCents            *int64     `json:",omitempty"`
	TotalCreditsCents           *int64     `json:",omitempty"`
	PaymentDueDate              *time.Time `json:",omitempty"`
	MinimumPaymentCents         *int64     `json:",omitempty"`
	PaymentToAvoidInterestCents *int64     `json:",omitempty"`
	CreditLimitCents            *int64     `json:",omitempty"`
	AvailableCreditCents        *int64     `json:",omitempty"`
}

// Transaction contains a single normalized movement.
type Transaction struct {
	PostedAt     time.Time
	Description  string
	Reference    string
	Direction    TransactionDirection
	AmountCents  int64
	BalanceCents *int64
}

// ParseResult captures the normalized statement plus best-effort diagnostics.
type ParseResult struct {
	Statement     Statement
	Warnings      []string
	Extraction    ExtractionDiagnostics
	ExtractedText string
}

// ExtractionDiagnostics exposes which extractor candidate won and what other
// extraction attempts were made along the way.
type ExtractionDiagnostics struct {
	SelectedExtractor string
	UsedRescue        bool
	Attempts          []TextExtractionAttempt
}
