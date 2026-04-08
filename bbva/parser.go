package bbva

import (
	edocuenta "github.com/DavidSerranoG/go-estado-cuenta-mx"
	internalbbva "github.com/DavidSerranoG/go-estado-cuenta-mx/internal/banks/bbva"
)

// Parser parses BBVA bank statements.
type Parser = internalbbva.Parser

var (
	_ edocuenta.Parser       = internalbbva.Parser{}
	_ edocuenta.ResultParser = internalbbva.Parser{}
)

// New returns a new BBVA parser.
func New() Parser {
	return internalbbva.New()
}
