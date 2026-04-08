package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/DavidSerranoG/go-estado-cuenta-mx/supported"
)

func main() {
	log.SetFlags(0)

	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./examples/basic <statement.pdf>")
		os.Exit(2)
	}

	pdfBytes, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	processor := supported.New()

	result, err := processor.ParsePDFResult(context.Background(), pdfBytes)
	if err != nil {
		log.Fatal(err)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		log.Fatal(err)
	}
}
