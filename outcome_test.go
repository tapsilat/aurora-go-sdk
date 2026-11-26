package aurora

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

const outcomeCreatedBody = `{
	"report_id": "rep-1",
	"transaction_id": "tx-root",
	"created": true,
	"changed": true,
	"amount_clamped": true,
	"resolution": "lifecycle_root",
	"label": {
		"label": "fraud",
		"tier": "external",
		"fraud_type": "stolen_card",
		"fraud_amount": "125.50",
		"currency": "TRY",
		"conflicting": true,
		"sources": ["chargeback", "review_case"]
	}
}`

func validOutcomeRequest() *OutcomeReportRequest {
	return &OutcomeReportRequest{
		Source:      OutcomeSourceChargeback,
		ExternalID:  "cb-1",
		Transaction: OutcomeTransactionRef{PaymentID: "pay-1"},
		Outcome:     OutcomeFraud,
		FraudType:   FraudTypeStolenCard,
	}
}

func TestOutcomeReportSendsFullBody(t *testing.T) {
	client, rec, _ := newTestServer(t, http.StatusCreated, outcomeCreatedBody)

	moved := false
	reported := time.Date(2026, 11, 1, 10, 30, 0, 0, time.UTC)
	resp, err := client.Outcomes.Report(context.Background(), &OutcomeReportRequest{
		Source:      OutcomeSourceChargeback,
		ExternalID:  "cb-1",
		Version:     2,
		Status:      OutcomeStatusActive,
		Transaction: OutcomeTransactionRef{TransactionID: "tx-1"},
		Outcome:     OutcomeFraud,
		FraudType:   FraudTypeStolenCard,
		ReasonCode:  "10.4",
		Amount:      "125.50",
		Currency:    "TRY",
		MoneyMoved:  &moved,
		ReportedAt:  reported,
		Note:        "issuer chargeback",
	})
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if rec.method != http.MethodPost || rec.path != "/api/v1/outcomes" {
		t.Errorf("request = %s %s, want POST /api/v1/outcomes", rec.method, rec.path)
	}

	want := map[string]any{
		"source":      "chargeback",
		"external_id": "cb-1",
		"version":     float64(2),
		"status":      "active",
		"transaction": map[string]any{"transaction_id": "tx-1"},
		"outcome":     "fraud",
		"fraud_type":  "stolen_card",
		"reason_code": "10.4",
		"amount":      "125.50",
		"currency":    "TRY",
		"money_moved": false,
		"reported_at": "2026-11-01T10:30:00Z",
		"note":        "issuer chargeback",
	}
	if len(rec.body) != len(want) {
		t.Errorf("body keys = %v, want %d keys", rec.body, len(want))
	}
	for key, value := range want {
		got := rec.body[key]
		if m, ok := value.(map[string]any); ok {
			gm, _ := got.(map[string]any)
			if len(gm) != len(m) || gm["transaction_id"] != m["transaction_id"] {
				t.Errorf("%s = %v, want %v", key, got, value)
			}
			continue
		}
		if got != value {
			t.Errorf("%s = %#v, want %#v", key, got, value)
		}
	}

	if resp.ReportID != "rep-1" || resp.TransactionID != "tx-root" || !resp.Created || !resp.Changed || !resp.AmountClamped {
		t.Errorf("response = %+v", resp)
	}
	if resp.Resolution != ResolutionLifecycleRoot {
		t.Errorf("Resolution = %q", resp.Resolution)
	}
	l := resp.Label
	if l == nil || l.Label != OutcomeFraud || l.Tier != LabelTierExternal || l.FraudType != FraudTypeStolenCard ||
		l.FraudAmount != "125.50" || l.Currency != "TRY" || !l.Conflicting || len(l.Sources) != 2 {
		t.Errorf("Label = %+v", l)
	}
}

// Optional fields left empty are not sent, so Aurora applies its defaults.
func TestOutcomeReportOmitsEmptyOptionalFields(t *testing.T) {
	client, rec, _ := newTestServer(t, http.StatusCreated, outcomeCreatedBody)

	if _, err := client.Outcomes.Report(context.Background(), validOutcomeRequest()); err != nil {
		t.Fatalf("Report: %v", err)
	}
	for _, key := range []string{"version", "status", "reason_code", "amount", "currency", "money_moved", "reported_at", "note"} {
		if _, ok := rec.body[key]; ok {
			t.Errorf("%s sent although empty: %v", key, rec.body[key])
		}
	}
	ref, _ := rec.body["transaction"].(map[string]any)
	if len(ref) != 1 || ref["payment_id"] != "pay-1" {
		t.Errorf("transaction = %v", rec.body["transaction"])
	}
}

func TestOutcomeReportUnchangedReturns200(t *testing.T) {
	client, _, _ := newTestServer(t, http.StatusOK,
		`{"report_id":"rep-1","transaction_id":"tx-1","created":false,"changed":false,"amount_clamped":false,"resolution":"exact","label":null}`)

	resp, err := client.Outcomes.Report(context.Background(), validOutcomeRequest())
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if resp.Created || resp.Changed || resp.Label != nil || resp.Resolution != ResolutionExact {
		t.Errorf("response = %+v", resp)
	}
}

func TestOutcomeReportRetract(t *testing.T) {
	client, rec, _ := newTestServer(t, http.StatusOK,
		`{"report_id":"rep-1","transaction_id":"tx-1","created":false,"changed":true,"amount_clamped":false,"resolution":"exact","label":null}`)

	resp, err := client.Outcomes.Report(context.Background(), &OutcomeReportRequest{
		Source:      OutcomeSourceChargeback,
		ExternalID:  "cb-1",
		Version:     3,
		Status:      OutcomeStatusRetracted,
		Transaction: OutcomeTransactionRef{PaymentID: "pay-1"},
	})
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if rec.body["status"] != "retracted" || rec.body["version"] != float64(3) {
		t.Errorf("body = %v", rec.body)
	}
	if _, ok := rec.body["outcome"]; ok {
		t.Errorf("outcome sent on retract: %v", rec.body)
	}
	if !resp.Changed || resp.Label != nil {
		t.Errorf("response = %+v", resp)
	}
}

func TestOutcomeReportErrorCodes(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		code       string
		sentinel   error
		candidates int
	}{
		{"not found", http.StatusNotFound,
			`{"error":"transaction not found","error_code":"transaction_not_found"}`,
			ErrorCodeTransactionNotFound, ErrTransactionNotFound, 0},
		{"ambiguous", http.StatusConflict,
			`{"error":"payment_id matches 3 transactions","error_code":"ambiguous_transaction","candidate_count":3}`,
			ErrorCodeAmbiguousTransaction, ErrAmbiguousTransaction, 3},
		{"external id conflict", http.StatusConflict,
			`{"error":"external_id is bound to another transaction","error_code":"external_id_bound_to_other_transaction"}`,
			ErrorCodeExternalIDConflict, ErrExternalIDConflict, 0},
		{"forbidden source", http.StatusForbidden,
			`{"error":"source is not allowed","error_code":"forbidden_source"}`,
			ErrorCodeForbiddenSource, nil, 0},
		{"currency mismatch", http.StatusBadRequest,
			`{"error":"currency does not match","error_code":"currency_mismatch"}`,
			ErrorCodeCurrencyMismatch, nil, 0},
	}
	sentinels := []error{ErrTransactionNotFound, ErrAmbiguousTransaction, ErrExternalIDConflict}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _, _ := newTestServer(t, tt.status, tt.body)
			_, err := client.Outcomes.Report(context.Background(), validOutcomeRequest())

			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("err = %v, want APIError", err)
			}
			if apiErr.StatusCode != tt.status || apiErr.Code != tt.code || apiErr.CandidateCount != tt.candidates {
				t.Errorf("got (%d, %q, %d), want (%d, %q, %d)",
					apiErr.StatusCode, apiErr.Code, apiErr.CandidateCount, tt.status, tt.code, tt.candidates)
			}
			for _, s := range sentinels {
				if got, want := errors.Is(err, s), s == tt.sentinel; got != want {
					t.Errorf("errors.Is(err, %v) = %v, want %v", s, got, want)
				}
			}
		})
	}
}

func TestOutcomeReportValidation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*OutcomeReportRequest) *OutcomeReportRequest
		field  string
	}{
		{"nil request", func(*OutcomeReportRequest) *OutcomeReportRequest { return nil }, "request"},
		{"empty source", func(r *OutcomeReportRequest) *OutcomeReportRequest { r.Source = ""; return r }, "source"},
		{"reserved source", func(r *OutcomeReportRequest) *OutcomeReportRequest { r.Source = "review_case"; return r }, "source"},
		{"blank external_id", func(r *OutcomeReportRequest) *OutcomeReportRequest { r.ExternalID = " "; return r }, "external_id"},
		{"long external_id", func(r *OutcomeReportRequest) *OutcomeReportRequest {
			r.ExternalID = strings.Repeat("x", 129)
			return r
		}, "external_id"},
		{"negative version", func(r *OutcomeReportRequest) *OutcomeReportRequest { r.Version = -1; return r }, "version"},
		{"unknown status", func(r *OutcomeReportRequest) *OutcomeReportRequest { r.Status = "deleted"; return r }, "status"},
		{"no transaction", func(r *OutcomeReportRequest) *OutcomeReportRequest {
			r.Transaction = OutcomeTransactionRef{}
			return r
		}, "transaction"},
		{"both transaction ids", func(r *OutcomeReportRequest) *OutcomeReportRequest {
			r.Transaction.TransactionID = "tx-1"
			return r
		}, "transaction"},
		{"blank transaction_id with payment_id", func(r *OutcomeReportRequest) *OutcomeReportRequest {
			r.Transaction = OutcomeTransactionRef{TransactionID: "  ", PaymentID: "pay-1"}
			return r
		}, "transaction.transaction_id"},
		{"blank payment_id with transaction_id", func(r *OutcomeReportRequest) *OutcomeReportRequest {
			r.Transaction = OutcomeTransactionRef{TransactionID: "tx-1", PaymentID: " "}
			return r
		}, "transaction.payment_id"},
		{"missing outcome", func(r *OutcomeReportRequest) *OutcomeReportRequest { r.Outcome = ""; return r }, "outcome"},
		{"unknown outcome", func(r *OutcomeReportRequest) *OutcomeReportRequest { r.Outcome = "chargeback"; return r }, "outcome"},
		{"fraud without type", func(r *OutcomeReportRequest) *OutcomeReportRequest { r.FraudType = ""; return r }, "fraud_type"},
		{"unknown fraud type", func(r *OutcomeReportRequest) *OutcomeReportRequest { r.FraudType = "phishing"; return r }, "fraud_type"},
		{"long reason_code", func(r *OutcomeReportRequest) *OutcomeReportRequest {
			r.ReasonCode = strings.Repeat("x", 33)
			return r
		}, "reason_code"},
		{"long note", func(r *OutcomeReportRequest) *OutcomeReportRequest {
			r.Note = strings.Repeat("x", 1001)
			return r
		}, "note"},
	}

	client, _, calls := newTestServer(t, http.StatusCreated, outcomeCreatedBody)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.Outcomes.Report(context.Background(), tt.mutate(validOutcomeRequest()))
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
	for _, mutate := range []func(*OutcomeReportRequest){
		func(r *OutcomeReportRequest) { r.Outcome, r.FraudType = OutcomeLegitimate, "" },
		func(r *OutcomeReportRequest) { r.Status, r.Outcome, r.FraudType = OutcomeStatusRetracted, "", "" },
		func(r *OutcomeReportRequest) { r.Source = OutcomeSourceOperator },
		func(r *OutcomeReportRequest) { r.ExternalID = strings.Repeat("ş", 128) },
		func(r *OutcomeReportRequest) { r.Note = strings.Repeat("ş", 1000) },
	} {
		req := validOutcomeRequest()
		mutate(req)
		if _, err := client.Outcomes.Report(context.Background(), req); err != nil {
			t.Errorf("request %+v rejected: %v", req, err)
		}
	}
}

func TestOutcomeReportMissingBaseURL(t *testing.T) {
	_, err := NewClient("key").Outcomes.Report(context.Background(), validOutcomeRequest())
	if !errors.Is(err, ErrMissingBaseURL) {
		t.Fatalf("err = %v, want ErrMissingBaseURL", err)
	}
}
