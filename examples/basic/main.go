package main

import (
	"context"
	"log"
	"os"

	"github.com/ledgermx/mxstatementpdf"
	"github.com/ledgermx/mxstatementpdf/hsbc"
)

func main() {
	pdfBytes, err := os.ReadFile("statement.pdf")
	if err != nil {
		log.Fatal(err)
	}

	processor := statementpdf.New(
		statementpdf.WithParser(hsbc.New()),
	)

	statement, err := processor.ParsePDF(context.Background(), pdfBytes)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("bank=%s account=%s tx=%d", statement.Bank, statement.AccountNumber, len(statement.Transactions))
}
