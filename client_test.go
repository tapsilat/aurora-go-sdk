package aurora

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// recordedRequest captures what the test server received.
type recordedRequest struct {
	method string
	path   string
	header http.Header
	body   map[string]any
}

// newTestServer returns a client pointed at a server that records each
// request and replies with the given status and body.
func newTestServer(t *testing.T, status int, body string) (*Client, *recordedRequest, *atomic.Int32) {
	t.Helper()
	rec := &recordedRequest{}
	calls := &atomic.Int32{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		rec.method = r.Method
		rec.path = r.URL.Path
		rec.header = r.Header.Clone()
		raw, _ := io.ReadAll(r.Body)
		rec.body = nil
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &rec.body); err != nil {
				t.Errorf("request body is not JSON: %v: %s", err, raw)
			}
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return NewClient("test-key", WithBaseURL(srv.URL+"/")), rec, calls
}

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

func TestCallbackTransaction(t *testing.T) {
	client, rec, _ := newTestServer(t, http.StatusOK, `{"message": "Callback received"}`)

	resp, err := client.Callback.Transaction(context.Background(), &CallbackTransactionRequest{
		PaymentID: "pay-1",
		Status:    "failed",
		Message:   "declined",
	})
	if err != nil {
		t.Fatalf("Transaction: %v", err)
	}
	if rec.path != "/api/v1/callback/transaction" {
		t.Errorf("path = %s", rec.path)
	}
	if rec.body["payment_id"] != "pay-1" || rec.body["status"] != "failed" {
		t.Errorf("body = %v", rec.body)
	}
	if resp.Message != "Callback received" {
		t.Errorf("Message = %q", resp.Message)
	}

}

func TestCallbackTransactionValidation(t *testing.T) {
	valid := func() *CallbackTransactionRequest {
		return &CallbackTransactionRequest{PaymentID: "pay-1", Status: CallbackStatusFailed, Message: "declined"}
	}
	tests := []struct {
		name   string
		mutate func(*CallbackTransactionRequest) *CallbackTransactionRequest
		field  string
	}{
		{"nil request", func(*CallbackTransactionRequest) *CallbackTransactionRequest { return nil }, "request"},
		{"empty payment_id", func(r *CallbackTransactionRequest) *CallbackTransactionRequest { r.PaymentID = ""; return r }, "payment_id"},
		{"blank payment_id", func(r *CallbackTransactionRequest) *CallbackTransactionRequest { r.PaymentID = "  "; return r }, "payment_id"},
		{"empty status", func(r *CallbackTransactionRequest) *CallbackTransactionRequest { r.Status = ""; return r }, "status"},
		{"unknown status", func(r *CallbackTransactionRequest) *CallbackTransactionRequest { r.Status = "ok"; return r }, "status"},
		{"status is case-sensitive", func(r *CallbackTransactionRequest) *CallbackTransactionRequest { r.Status = "SUCCESS"; return r }, "status"},
		{"empty message", func(r *CallbackTransactionRequest) *CallbackTransactionRequest { r.Message = ""; return r }, "message"},
	}

	client, _, calls := newTestServer(t, http.StatusOK, `{"message": "Callback received"}`)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.Callback.Transaction(context.Background(), tt.mutate(valid()))
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

	// Aurora accepts these; the SDK must not be stricter than the API.
	for _, req := range []*CallbackTransactionRequest{
		{PaymentID: "pay-1", Status: CallbackStatusSuccess, Message: "  "},
		{PaymentID: "pay-1", Status: CallbackStatusSuccess, Message: "paid"},
	} {
		if _, err := client.Callback.Transaction(context.Background(), req); err != nil {
			t.Errorf("request %+v rejected: %v", req, err)
		}
	}
}

func TestAPIErrorParsing(t *testing.T) {
	long := strings.Repeat("x", maxErrorMessageLen+100)
	tests := []struct {
		name    string
		status  int
		body    string
		message string
	}{
		{"error key", http.StatusBadRequest, `{"error": "bad rule"}`, "bad rule"},
		{"message key", http.StatusBadRequest, `{"message": "status is required"}`, "status is required"},
		{"plain text", http.StatusBadGateway, "upstream down", "upstream down"},
		{"empty body", http.StatusServiceUnavailable, "", http.StatusText(http.StatusServiceUnavailable)},
		{"long body", http.StatusInternalServerError, long, long[:maxErrorMessageLen] + "..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _, _ := newTestServer(t, tt.status, tt.body)
			_, err := client.Process.Execute(context.Background(), &ProcessRequest{
				RuleID:      "r",
				Transaction: &Transaction{Type: TransactionTypeCard},
			})
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("err = %v, want APIError", err)
			}
			if apiErr.StatusCode != tt.status || apiErr.Message != tt.message {
				t.Errorf("got (%d, %q), want (%d, %q)", apiErr.StatusCode, apiErr.Message, tt.status, tt.message)
			}
			if string(apiErr.Body) != tt.body {
				t.Errorf("Body = %q, want %q", apiErr.Body, tt.body)
			}
		})
	}
}

func TestErrorHelpers(t *testing.T) {
	wrap := func(status int) error { return errors.Join(errors.New("ctx"), &APIError{StatusCode: status}) }

	if !IsRateLimited(wrap(http.StatusTooManyRequests)) || IsRateLimited(wrap(http.StatusBadRequest)) {
		t.Error("IsRateLimited")
	}
	if !IsNotFound(wrap(http.StatusNotFound)) || IsNotFound(errors.New("plain")) {
		t.Error("IsNotFound")
	}
	if !IsUnauthorized(wrap(http.StatusUnauthorized)) || !IsUnauthorized(wrap(http.StatusForbidden)) || IsUnauthorized(nil) {
		t.Error("IsUnauthorized")
	}
}

func TestMissingBaseURL(t *testing.T) {
	client := NewClient("key")
	_, err := client.Process.Execute(context.Background(), &ProcessRequest{
		RuleID:      "r",
		Transaction: &Transaction{Type: TransactionTypeCard},
	})
	if !errors.Is(err, ErrMissingBaseURL) {
		t.Fatalf("err = %v, want ErrMissingBaseURL", err)
	}
}

func TestContextCancellation(t *testing.T) {
	client, _, _ := newTestServer(t, http.StatusOK, `{}`)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Process.Execute(ctx, &ProcessRequest{
		RuleID:      "r",
		Transaction: &Transaction{Type: TransactionTypeCard},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}
