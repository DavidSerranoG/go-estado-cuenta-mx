package hsbc

import (
	edocuenta "github.com/DavidSerranoG/go-estado-cuenta-mx"
	internalhsbc "github.com/DavidSerranoG/go-estado-cuenta-mx/internal/banks/hsbc"
)

// Parser parses HSBC bank statements.
type Parser = internalhsbc.Parser

var (
	_ edocuenta.Parser       = internalhsbc.Parser{}
	_ edocuenta.ResultParser = internalhsbc.Parser{}
)

// New returns a new HSBC parser.
func New() Parser {
	return internalhsbc.New()
}
