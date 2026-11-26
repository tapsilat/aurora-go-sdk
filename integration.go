package aurora

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// IntegrationService tells Aurora what an integration sends, so rules can be
// checked against it before any real transaction arrives.
type IntegrationService struct {
	client *Client
}

// AbsentField is a field a flow never carries, and why. Aurora then tells a
// rule that needs it is not meant for this flow, instead of reporting
// missing data.
type AbsentField struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

// IntegrationSample is one kind of transaction the integration sends, built
// the way it builds real ones.
type IntegrationSample struct {
	// Flow names the kind, e.g. "bank_withdraw"; Variant tells samples of one
	// flow apart, e.g. a currency. Flow and Variant together must be unique.
	Flow    string `json:"flow"`
	Variant string `json:"variant,omitempty"`
	// Title is shown to users, e.g. "Banka çekimi".
	Title string `json:"title,omitempty"`
	// FixedFields lists fields whose value is the same on every transaction
	// of this flow (a bank withdrawal always has payment_method
	// "bank_withdraw"). Other fields with a value are taken to vary, and
	// fields left empty are taken to be never sent. The transaction type is
	// always fixed.
	FixedFields []string `json:"fixed_fields,omitempty"`
	// Absent lists fields this flow never carries by its nature, with why.
	// They must be empty in Transaction.
	Absent []AbsentField `json:"absent,omitempty"`
	// Transaction is a sample as it would be sent to Process.Execute.
	Transaction *Transaction `json:"transaction"`
}

// PublishSamplesRequest replaces every sample Source published before.
type PublishSamplesRequest struct {
	// Source names the integration, e.g. "admin".
	Source  string              `json:"source"`
	Samples []IntegrationSample `json:"samples"`
}

// PublishSamplesResponse reports what Aurora stored.
type PublishSamplesResponse struct {
	Source string `json:"source"`
	Stored int    `json:"stored"`
}

// PublishSamples replaces the samples of req.Source for the organization the
// API key belongs to (PUT /integration/samples). Publishing an empty list
// removes them. Aurora's rule self-test then reports, for every rule, on which
// of these flows it can act and why not on the others.
func (s *IntegrationService) PublishSamples(ctx context.Context, req *PublishSamplesRequest) (*PublishSamplesResponse, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	httpReq, err := s.client.newRequest(ctx, http.MethodPut, pathIntegrationSamples, req)
	if err != nil {
		return nil, err
	}

	var resp PublishSamplesResponse
	if err := s.client.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Identity is who Aurora takes the API key's requests to come from.
type Identity struct {
	Organization IdentityOrganization `json:"organization"`
	// Token is the API key itself; nil when Aurora did not recognize the
	// request as coming from an API key.
	Token *IdentityToken `json:"token"`
}

// IdentityOrganization is the organization whose rules score the key's
// transactions and where its reports are stored.
type IdentityOrganization struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Title string `json:"title"`
}

// IdentityToken describes the API key.
type IdentityToken struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	ExpiresAt *time.Time `json:"expires_at"`
}

// Identity returns the organization and key the client authenticates as
// (GET /integration/identity). Call it to check a key before relying on it:
// an invalid or expired key returns an *APIError for which IsUnauthorized is
// true, and a key of another organization shows up in Organization.
func (s *IntegrationService) Identity(ctx context.Context) (*Identity, error) {
	httpReq, err := s.client.newRequest(ctx, http.MethodGet, pathIntegrationIdentity, nil)
	if err != nil {
		return nil, err
	}

	var resp Identity
	if err := s.client.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Connection statuses for ConnectionRequest.Status.
const (
	ConnectionStatusActive  = "active"
	ConnectionStatusRemoved = "removed"
)

// ConnectionRequest reports how the integration uses the API key, for example
// a fraud provider set up with it and the ruleset it sends.
type ConnectionRequest struct {
	// Source names the integration, such as "admin". Required.
	Source string `json:"source"`
	// Reference is the integration's ID of the setup, such as the fraud
	// provider ID. Required.
	Reference string `json:"reference"`
	Name      string `json:"name,omitempty"`
	// RulesetID is the ruleset the integration sends with Process.
	RulesetID string `json:"ruleset_id,omitempty"`
	Enabled   bool   `json:"enabled"`
	CanReject bool   `json:"can_reject"`
	// Status is ConnectionStatusActive (default) or ConnectionStatusRemoved
	// when the setup was deleted.
	Status string `json:"status,omitempty"`
	// DryRun checks the key and ruleset without Aurora storing anything.
	DryRun bool `json:"dry_run,omitempty"`
}

// ConnectionResponse is the key's identity and whether the ruleset exists in
// its organization.
type ConnectionResponse struct {
	Identity
	Ruleset RulesetCheck `json:"ruleset"`
	// Stored is false for a dry run and a removed setup.
	Stored bool `json:"stored"`
}

// RulesetCheck tells whether a ruleset ID names a ruleset of the key's
// organization. Process fails for a ruleset that is not found.
type RulesetCheck struct {
	ID    string `json:"id"`
	Name  string `json:"name,omitempty"`
	Found bool   `json:"found"`
}

// ReportConnection tells Aurora how the integration uses the key (POST
// /integration/connection), so Aurora can show it under the key, and returns
// the key's identity and the ruleset check. Errors are as for Identity.
func (s *IntegrationService) ReportConnection(ctx context.Context, req *ConnectionRequest) (*ConnectionResponse, error) {
	if req == nil {
		return nil, &ValidationError{Field: "request", Message: "request is required"}
	}
	if strings.TrimSpace(req.Source) == "" {
		return nil, &ValidationError{Field: "source", Message: "source is required"}
	}
	if strings.TrimSpace(req.Reference) == "" {
		return nil, &ValidationError{Field: "reference", Message: "reference is required"}
	}
	if req.Status != "" && req.Status != ConnectionStatusActive && req.Status != ConnectionStatusRemoved {
		return nil, &ValidationError{Field: "status", Message: "status must be active or removed"}
	}

	httpReq, err := s.client.newRequest(ctx, http.MethodPost, pathIntegrationConnect, req)
	if err != nil {
		return nil, err
	}

	var resp ConnectionResponse
	if err := s.client.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (req *PublishSamplesRequest) validate() error {
	if req == nil {
		return &ValidationError{Field: "request", Message: "request is required"}
	}
	if strings.TrimSpace(req.Source) == "" {
		return &ValidationError{Field: "source", Message: "source is required"}
	}

	type sampleKey struct{ flow, variant string }
	seen := make(map[sampleKey]bool, len(req.Samples))
	for i, sample := range req.Samples {
		prefix := fmt.Sprintf("samples[%d]", i)
		if strings.TrimSpace(sample.Flow) == "" {
			return &ValidationError{Field: prefix + ".flow", Message: "flow is required"}
		}
		if sample.Transaction == nil || !validTransactionTypes[sample.Transaction.Type] {
			return &ValidationError{Field: prefix + ".transaction.type", Message: "invalid or missing transaction type"}
		}
		absent := make(map[string]bool, len(sample.Absent))
		for j, a := range sample.Absent {
			field := fmt.Sprintf("%s.absent[%d]", prefix, j)
			switch {
			case strings.TrimSpace(a.Field) == "":
				return &ValidationError{Field: field + ".field", Message: "field is required"}
			case !validAbsentReasons[a.Reason]:
				return &ValidationError{Field: field + ".reason", Message: fmt.Sprintf("unknown reason %q", a.Reason)}
			case absent[a.Field]:
				return &ValidationError{Field: field, Message: fmt.Sprintf("field %q appears twice", a.Field)}
			}
			absent[a.Field] = true
		}
		key := sampleKey{sample.Flow, sample.Variant}
		if seen[key] {
			return &ValidationError{Field: prefix, Message: fmt.Sprintf("flow %q variant %q appears twice", sample.Flow, sample.Variant)}
		}
		seen[key] = true
	}
	return nil
}
