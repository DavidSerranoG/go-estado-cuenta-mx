package statementpdf

import "errors"

var (
	ErrEmptyPDF            = errors.New("statementpdf: empty pdf payload")
	ErrNoParsersConfigured = errors.New("statementpdf: no parsers configured")
	ErrUnsupportedFormat   = errors.New("statementpdf: unsupported statement format")
	ErrUnsupportedBank     = errors.New("statementpdf: unsupported bank")
)
