package aurora

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestProcessExecuteSendsRequestAndDecodesResponse(t *testing.T) {
	client, rec, _ := newTestServer(t, http.StatusOK, `{
		"transaction_id": "tx-1",
		"challenge": true,
		"challenge_message": "verify with otp",
		"challenge_type": "otp",
		"score": 40,
		"execution_time": 1500000,
		"warnings": [{"code": "card_holder_name_mismatch", "message": "name differs"}],
		"traces": [{"node_id": "n1", "node_type": "condition", "title": "Amount", "result": "true", "score_added": 40, "matched": true, "duration": 1000}],
		"shadow_results": [{"rule_id": "r2", "rule_name": "Shadow", "rejected": true, "score": 90}]
	}`)

	resp, err := client.Process.Execute(context.Background(), &ProcessRequest{
		RulesetID:   "ruleset-1",
		Transaction: &Transaction{Type: TransactionTypeCard, Common: &Common{Amount: 10}},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if rec.method != http.MethodPost || rec.path != "/api/v1/process" {
		t.Errorf("request = %s %s, want POST /api/v1/process", rec.method, rec.path)
	}
	if got := rec.header.Get("Authorization"); got != "Bearer test-key" {
		t.Errorf("Authorization = %q", got)
	}
	if got := rec.header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q", got)
	}
	if rec.body["ruleset_id"] != "ruleset-1" {
		t.Errorf("ruleset_id not sent: %v", rec.body)
	}

	if !resp.Challenge || resp.ChallengeMessage != "verify with otp" || resp.ChallengeType != "otp" {
		t.Errorf("challenge not decoded: %+v", resp)
	}
	if resp.Allow || resp.Rejected || resp.NeedInspect {
		t.Errorf("unexpected decision flags: %+v", resp)
	}
	if resp.ExecutionTime != 1500*time.Microsecond {
		t.Errorf("ExecutionTime = %v", resp.ExecutionTime)
	}
	if len(resp.Warnings) != 1 || resp.Warnings[0].Code != "card_holder_name_mismatch" {
		t.Errorf("warnings = %+v", resp.Warnings)
	}

	// Traces and shadow results carry raw field values (emails, IBANs).
	// Callers log and persist the response, so the SDK must keep dropping them.
	encoded, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	for _, key := range []string{`"traces"`, `"shadow_results"`} {
		if strings.Contains(string(encoded), key) {
			t.Errorf("response should not carry %s: %s", key, encoded)
		}
	}
}

func TestProcessExecuteReevaluatesByTransactionID(t *testing.T) {
	client, rec, calls := newTestServer(t, http.StatusOK, `{"allow": true}`)

	resp, err := client.Process.Execute(context.Background(), &ProcessRequest{
		RulesetID:     "ruleset-1",
		TransactionID: "0b0c6a9e-8f1d-4d6e-9a43-0c8f7d2b1e55",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if calls.Load() != 1 || !resp.Allow {
		t.Fatalf("calls = %d, resp = %+v", calls.Load(), resp)
	}
	if _, ok := rec.body["transaction"]; ok {
		t.Errorf("transaction should be omitted: %v", rec.body)
	}
	if rec.body["transaction_id"] != "0b0c6a9e-8f1d-4d6e-9a43-0c8f7d2b1e55" {
		t.Errorf("transaction_id not sent: %v", rec.body)
	}
}

func TestProcessExecuteValidation(t *testing.T) {
	tests := []struct {
		name  string
		req   *ProcessRequest
		field string
	}{
		{"nil request", nil, "request"},
		{"no rule or ruleset", &ProcessRequest{Transaction: &Transaction{Type: TransactionTypeCard}}, "rule_id"},
		{"no transaction", &ProcessRequest{RuleID: "r"}, "transaction"},
		{"missing type", &ProcessRequest{RuleID: "r", Transaction: &Transaction{}}, "transaction.type"},
		{"unknown type", &ProcessRequest{RuleID: "r", Transaction: &Transaction{Type: "wire"}}, "transaction.type"},
		{"unknown type on re-evaluation", &ProcessRequest{RuleID: "r", TransactionID: "tx", Transaction: &Transaction{Type: "wire"}}, "transaction.type"},
	}

	client, _, calls := newTestServer(t, http.StatusOK, `{}`)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.Process.Execute(context.Background(), tt.req)
			var vErr *ValidationError
			if !errors.As(err, &vErr) {
				t.Fatalf("err = %v, want ValidationError", err)
			}
			if vErr.Field != tt.field {
				t.Errorf("Field = %q, want %q", vErr.Field, tt.field)
			}
		})
	}
	if calls.Load() != 0 {
		t.Errorf("invalid requests reached the server %d times", calls.Load())
	}
}

// Aurora uses rule_id when both are set; the SDK sends both unchanged and
// leaves that choice to the server.
func TestProcessExecuteSendsBothRuleAndRuleset(t *testing.T) {
	client, rec, _ := newTestServer(t, http.StatusOK, `{"allow": true}`)
	_, err := client.Process.Execute(context.Background(), &ProcessRequest{
		RuleID:      "rule-1",
		RulesetID:   "ruleset-1",
		Transaction: &Transaction{Type: TransactionTypeCard},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.body["rule_id"] != "rule-1" || rec.body["ruleset_id"] != "ruleset-1" {
		t.Errorf("body = %v", rec.body)
	}
}

func TestProcessExecuteReevaluatesWithEmptyTransaction(t *testing.T) {
	client, rec, calls := newTestServer(t, http.StatusOK, `{"allow": true}`)
	_, err := client.Process.Execute(context.Background(), &ProcessRequest{
		RulesetID:     "ruleset-1",
		TransactionID: "0b0c6a9e-8f1d-4d6e-9a43-0c8f7d2b1e55",
		Transaction:   &Transaction{},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if calls.Load() != 1 || rec.body["transaction_id"] == nil {
		t.Errorf("calls = %d, body = %v", calls.Load(), rec.body)
	}
}

// Names keep Turkish characters byte for byte, and amounts are written as
// the shortest decimal that round-trips, so 0.1 is sent as 0.1.
func TestProcessExecuteEncodesTextAndAmounts(t *testing.T) {
	var raw []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ = io.ReadAll(r.Body)
		_, _ = io.WriteString(w, `{"allow": true}`)
	}))
	defer srv.Close()
	client := NewClient("key", WithBaseURL(srv.URL))

	tests := []struct {
		amount float64
		want   string
	}{
		{0.1, `"amount":0.1`},
		{1500.5, `"amount":1500.5`},
		{1234567.89, `"amount":1234567.89`},
		{999999999999.99, `"amount":999999999999.99`},
	}
	for _, tt := range tests {
		_, err := client.Process.Execute(context.Background(), &ProcessRequest{
			RuleID: "r",
			Transaction: &Transaction{
				Type:   TransactionTypeCard,
				Common: &Common{Amount: tt.amount, FirstName: "Şükrü", LastName: "Çağlar Öztürk"},
				Card:   &Card{Holder: "ŞÜKRÜ ÇAĞLAR ÖZTÜRK"},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		body := string(raw)
		for _, want := range []string{tt.want, `"first_name":"Şükrü"`, `"last_name":"Çağlar Öztürk"`, `"holder":"ŞÜKRÜ ÇAĞLAR ÖZTÜRK"`} {
			if !strings.Contains(body, want) {
				t.Errorf("body missing %s: %s", want, body)
			}
		}
	}
}
