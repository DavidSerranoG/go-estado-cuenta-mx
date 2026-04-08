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

// TransactionKind classifies the normalized transaction direction.
type TransactionKind string

const (
	TransactionKindDebit  TransactionKind = "debit"
	TransactionKindCredit TransactionKind = "credit"
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
	Transactions  []Transaction
}

// Transaction contains a single normalized movement.
type Transaction struct {
	PostedAt     time.Time
	Description  string
	Reference    string
	Kind         TransactionKind
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
