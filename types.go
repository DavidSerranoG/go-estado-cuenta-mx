package statementpdf

import "time"

// Statement contains normalized data extracted from a bank statement PDF.
type Statement struct {
	Bank          string
	AccountNumber string
	Currency      string
	PeriodStart   time.Time
	PeriodEnd     time.Time
	Transactions  []Transaction
	Warnings      []string
	ExtractedText string
}

// Transaction contains a single normalized account movement.
type Transaction struct {
	PostedAt     time.Time
	Description  string
	Reference    string
	Type         string
	AmountCents  int64
	BalanceCents *int64
	RawLine      string
}
