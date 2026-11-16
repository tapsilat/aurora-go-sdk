package aurora

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestPublishSamplesSendsRequestAndDecodesResponse(t *testing.T) {
	client, rec, _ := newTestServer(t, http.StatusOK, `{"source":"admin","stored":1}`)

	resp, err := client.Integration.PublishSamples(context.Background(), &PublishSamplesRequest{
		Source: "admin",
		Samples: []IntegrationSample{{
			Flow:        "bank_withdraw",
			Title:       "Banka çekimi",
			FixedFields: []string{FieldTransactionType, FieldPaymentMethod},
			Transaction: &Transaction{Type: TransactionTypeWithdrawal, Common: &Common{Amount: 100}},
		}},
	})
	if err != nil {
		t.Fatalf("PublishSamples: %v", err)
	}
	if rec.method != http.MethodPut || rec.path != "/api/v1/integration/samples" {
		t.Errorf("request = %s %s, want PUT /api/v1/integration/samples", rec.method, rec.path)
	}
	samples, _ := rec.body["samples"].([]any)
	if rec.body["source"] != "admin" || len(samples) != 1 {
		t.Fatalf("body = %v", rec.body)
	}
	sample := samples[0].(map[string]any)
	if sample["flow"] != "bank_withdraw" || sample["transaction"].(map[string]any)["type"] != "withdrawal" {
		t.Errorf("sample = %v", sample)
	}
	if resp.Source != "admin" || resp.Stored != 1 {
		t.Errorf("response = %+v", resp)
	}
}

func TestPublishSamplesValidates(t *testing.T) {
	client, _, calls := newTestServer(t, http.StatusOK, `{}`)
	withdrawal := &Transaction{Type: TransactionTypeWithdrawal}
	cases := map[string]*PublishSamplesRequest{
		"nil request": nil,
		"no source":   {Samples: []IntegrationSample{{Flow: "a", Transaction: withdrawal}}},
		"no flow":     {Source: "admin", Samples: []IntegrationSample{{Transaction: withdrawal}}},
		"no tx":       {Source: "admin", Samples: []IntegrationSample{{Flow: "a"}}},
		"bad type":    {Source: "admin", Samples: []IntegrationSample{{Flow: "a", Transaction: &Transaction{Type: "nope"}}}},
		"duplicate":   {Source: "admin", Samples: []IntegrationSample{{Flow: "a", Transaction: withdrawal}, {Flow: "a", Transaction: withdrawal}}},
		"absent without field": {Source: "admin", Samples: []IntegrationSample{{Flow: "a", Transaction: withdrawal,
			Absent: []AbsentField{{Reason: AbsentNotAMember}}}}},
		"absent unknown reason": {Source: "admin", Samples: []IntegrationSample{{Flow: "a", Transaction: withdrawal,
			Absent: []AbsentField{{Field: FieldAccountAgeDays, Reason: "because"}}}}},
		"absent twice": {Source: "admin", Samples: []IntegrationSample{{Flow: "a", Transaction: withdrawal,
			Absent: []AbsentField{{Field: FieldIPAddress, Reason: AbsentNoSession}, {Field: FieldIPAddress, Reason: AbsentNoSession}}}}},
	}
	for name, req := range cases {
		_, err := client.Integration.PublishSamples(context.Background(), req)
		var verr *ValidationError
		if !errors.As(err, &verr) {
			t.Errorf("%s: err = %v, want a ValidationError", name, err)
		}
	}
	if calls.Load() != 0 {
		t.Errorf("invalid requests reached the server %d times", calls.Load())
	}

	// An empty list is valid: it removes the source's samples.
	if _, err := client.Integration.PublishSamples(context.Background(), &PublishSamplesRequest{Source: "admin"}); err != nil {
		t.Errorf("empty publish: %v", err)
	}
}
