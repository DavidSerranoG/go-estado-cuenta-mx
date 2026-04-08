package edocuenta_test

import (
	"encoding/json"
	"testing"

	edocuenta "github.com/DavidSerranoG/go-estado-cuenta-mx"
)

func TestStatementJSONOmitsUnknownAccountClassAndNilSummary(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(edocuenta.Statement{
		Bank:         edocuenta.BankBBVA,
		Transactions: []edocuenta.Transaction{},
	})
	if err != nil {
		t.Fatalf("marshal statement: %v", err)
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal statement: %v", err)
	}

	if _, ok := decoded["AccountClass"]; ok {
		t.Fatalf("expected AccountClass to be omitted, got %s", data)
	}
	if _, ok := decoded["Summary"]; ok {
		t.Fatalf("expected Summary to be omitted, got %s", data)
	}
}

func TestStatementJSONUsesDirectionAndOmitsUnsetSummaryFields(t *testing.T) {
	t.Parallel()

	opening := int64(12345)
	data, err := json.Marshal(edocuenta.Statement{
		Bank:         edocuenta.BankHSBC,
		AccountClass: edocuenta.AccountClassAsset,
		Summary: &edocuenta.StatementSummary{
			OpeningBalanceCents: &opening,
		},
		Transactions: []edocuenta.Transaction{
			{
				Description: "deposit",
				Direction:   edocuenta.TransactionDirectionCredit,
				AmountCents: 12345,
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal statement: %v", err)
	}

	var decoded struct {
		AccountClass string
		Summary      map[string]json.RawMessage
		Transactions []map[string]json.RawMessage
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal statement: %v", err)
	}

	if decoded.AccountClass != "asset" {
		t.Fatalf("unexpected account class %q", decoded.AccountClass)
	}
	if _, ok := decoded.Summary["OpeningBalanceCents"]; !ok {
		t.Fatalf("expected OpeningBalanceCents in summary, got %s", data)
	}
	if _, ok := decoded.Summary["ClosingBalanceCents"]; ok {
		t.Fatalf("expected ClosingBalanceCents to be omitted, got %s", data)
	}
	if _, ok := decoded.Transactions[0]["Direction"]; !ok {
		t.Fatalf("expected Direction field in transaction, got %s", data)
	}
	if _, ok := decoded.Transactions[0]["Kind"]; ok {
		t.Fatalf("did not expect legacy Kind field, got %s", data)
	}
}
