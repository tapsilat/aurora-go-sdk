package aurora

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

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

func TestCallbackTransactionAPIError(t *testing.T) {
	client, _, _ := newTestServer(t, http.StatusInternalServerError, `{"message": "payment outcome is already final: success"}`)

	_, err := client.Callback.Transaction(context.Background(), &CallbackTransactionRequest{
		PaymentID: "pay-1",
		Status:    CallbackStatusFailed,
		Message:   "declined",
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want APIError", err)
	}
	if apiErr.StatusCode != http.StatusInternalServerError || apiErr.Message != "payment outcome is already final: success" {
		t.Errorf("got (%d, %q)", apiErr.StatusCode, apiErr.Message)
	}
}

func TestCallbackTransactionMissingBaseURL(t *testing.T) {
	_, err := NewClient("key").Callback.Transaction(context.Background(), &CallbackTransactionRequest{
		PaymentID: "pay-1",
		Status:    CallbackStatusSuccess,
		Message:   "paid",
	})
	if !errors.Is(err, ErrMissingBaseURL) {
		t.Fatalf("err = %v, want ErrMissingBaseURL", err)
	}
}
