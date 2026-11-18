package aurora

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestTransactionAttributesJSON(t *testing.T) {
	payload, err := json.Marshal(Transaction{Type: TransactionTypeCard, Attributes: map[string]string{"check_in_date": "2026-12-20"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), `"attributes":{"check_in_date":"2026-12-20"}`) {
		t.Fatalf("attributes missing: %s", payload)
	}
	payload, err = json.Marshal(Transaction{Type: TransactionTypeCard})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), `"attributes"`) {
		t.Fatalf("unset attributes must be omitted: %s", payload)
	}
}

func TestProcessRequestValidatesAttributes(t *testing.T) {
	request := func(attributes map[string]string) *ProcessRequest {
		return &ProcessRequest{RulesetID: "rs", Transaction: &Transaction{Type: TransactionTypeCard, Attributes: attributes}}
	}
	if err := request(map[string]string{"vehicle_class": "suv"}).validate(); err != nil {
		t.Fatalf("valid attributes: %v", err)
	}
	tooMany := map[string]string{}
	for i := range MaxAttributes + 1 {
		tooMany["k"+strings.Repeat("a", i)] = "v"
	}
	for name, attributes := range map[string]map[string]string{
		"upper case key": {"CheckIn": "x"},
		"quoted key":     {"x'": "x"},
		"long value":     {"note": strings.Repeat("a", MaxAttributeValueLength+1)},
		"too many":       tooMany,
	} {
		var validation *ValidationError
		if err := request(attributes).validate(); !errors.As(err, &validation) || validation.Field != "transaction.attributes" {
			t.Errorf("%s: got %v", name, err)
		}
	}
}
