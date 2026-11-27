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

func TestIdentityDecodesOrganizationAndToken(t *testing.T) {
	client, rec, _ := newTestServer(t, http.StatusOK, `{"organization":{"id":"org-1","name":"garanti-gold","title":"Garanti Gold"},"token":{"id":"tok-1","name":"Tapsilat Prod","expires_at":"2027-07-10T07:45:20Z"}}`)

	identity, err := client.Integration.Identity(context.Background())
	if err != nil {
		t.Fatalf("Identity: %v", err)
	}
	if rec.method != http.MethodGet || rec.path != "/api/v1/integration/identity" {
		t.Errorf("request = %s %s, want GET /api/v1/integration/identity", rec.method, rec.path)
	}
	if identity.Organization.ID != "org-1" || identity.Organization.Title != "Garanti Gold" {
		t.Errorf("organization = %+v", identity.Organization)
	}
	if identity.Token == nil || identity.Token.Name != "Tapsilat Prod" || identity.Token.ExpiresAt == nil || identity.Token.ExpiresAt.Year() != 2027 {
		t.Errorf("token = %+v", identity.Token)
	}
}

func TestIdentityUnauthorized(t *testing.T) {
	client, _, _ := newTestServer(t, http.StatusUnauthorized, `{"code":401,"message":"Authorization Required"}`)

	_, err := client.Integration.Identity(context.Background())
	if !IsUnauthorized(err) {
		t.Fatalf("err = %v, want unauthorized", err)
	}
}

func TestReportConnectionSendsRequestAndDecodesResponse(t *testing.T) {
	client, rec, _ := newTestServer(t, http.StatusOK, `{"organization":{"id":"org-1","name":"garanti-gold","title":"Garanti Gold"},"token":{"id":"tok-1","name":"Tapsilat Prod"},"ruleset":{"id":"rs-1","name":"Garanti Gold Ruleset","found":true},"stored":true}`)

	resp, err := client.Integration.ReportConnection(context.Background(), &ConnectionRequest{
		Source: "admin", Reference: "provider-1", Name: "GarantiGold Prod", RulesetID: "rs-1", Enabled: true,
	})
	if err != nil {
		t.Fatalf("ReportConnection: %v", err)
	}
	if rec.method != http.MethodPost || rec.path != "/api/v1/integration/connection" {
		t.Errorf("request = %s %s", rec.method, rec.path)
	}
	if rec.body["source"] != "admin" || rec.body["reference"] != "provider-1" || rec.body["ruleset_id"] != "rs-1" || rec.body["enabled"] != true || rec.body["can_reject"] != false {
		t.Errorf("body = %v", rec.body)
	}
	if _, ok := rec.body["dry_run"]; ok {
		t.Errorf("dry_run sent when false: %v", rec.body)
	}
	if !resp.Ruleset.Found || resp.Ruleset.Name != "Garanti Gold Ruleset" || !resp.Stored || resp.Organization.Title != "Garanti Gold" || resp.Token.Name != "Tapsilat Prod" {
		t.Errorf("response = %+v", resp)
	}
}

func TestReportConnectionValidates(t *testing.T) {
	client, _, calls := newTestServer(t, http.StatusOK, `{}`)
	for _, req := range []*ConnectionRequest{nil, {Reference: "r"}, {Source: "admin"}, {Source: "admin", Reference: "r", Status: "paused"}} {
		var validation *ValidationError
		if _, err := client.Integration.ReportConnection(context.Background(), req); !errors.As(err, &validation) {
			t.Errorf("%+v: err = %v, want a validation error", req, err)
		}
	}
	if calls.Load() != 0 {
		t.Errorf("invalid requests reached the server %d times", calls.Load())
	}
}
