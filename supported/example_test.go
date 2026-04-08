package supported_test

import (
	"context"
	"fmt"

	edocuenta "github.com/DavidSerranoG/go-estado-cuenta-mx"
	"github.com/DavidSerranoG/go-estado-cuenta-mx/supported"
)

func ExampleNew() {
	processor := supported.New(
		edocuenta.WithExtractor(staticExtractor{text: sampleHSBCText}),
	)

	statement, err := processor.ParsePDF(context.Background(), []byte("pdf"))
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("%s %s %d\n", statement.Bank, statement.Currency, len(statement.Transactions))
	// Output:
	// hsbc MXN 2
}
