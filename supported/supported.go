package supported

import (
	edocuenta "github.com/DavidSerranoG/go-estado-cuenta-mx"
	"github.com/DavidSerranoG/go-estado-cuenta-mx/bbva"
	"github.com/DavidSerranoG/go-estado-cuenta-mx/hsbc"
)

// Parsers returns the built-in parsers supported by this module.
func Parsers() []edocuenta.Parser {
	return []edocuenta.Parser{
		bbva.New(),
		hsbc.New(),
	}
}

// New builds a processor preloaded with the built-in parsers.
func New(opts ...edocuenta.Option) *edocuenta.Processor {
	base := []edocuenta.Option{
		edocuenta.WithParsers(Parsers()...),
	}

	return edocuenta.New(append(base, opts...)...)
}
