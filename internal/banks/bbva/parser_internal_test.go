package bbva

import (
	"testing"

	edocuenta "github.com/DavidSerranoG/go-estado-cuenta-mx"
)

func TestClassifyByDescriptionResolvesKnownStatementPatterns(t *testing.T) {
	t.Parallel()

	cases := map[string]edocuenta.TransactionDirection{
		"ABONO POR CORRECCION WWW ALIEXPRESS COM":   edocuenta.TransactionDirectionCredit,
		"PAGO CUENTA DE TERCERO BNET 0105982138":    edocuenta.TransactionDirectionDebit,
		"ADYENMX*UBER EATS RFC: UPM 200220LK5":      edocuenta.TransactionDirectionDebit,
		"AMAZON MX MARKETPLACE RFC: ANE 140618P37":  edocuenta.TransactionDirectionDebit,
		"AMAZON MX RFC: ANE 140618P37":              edocuenta.TransactionDirectionDebit,
	}

	for description, want := range cases {
		if got := classifyByDescription(description); got != want {
			t.Fatalf("classifyByDescription(%q) = %q, want %q", description, got, want)
		}
	}
}
