package aurora

import (
	"context"
	"net/http"
)

// ProcessService handles rule processing operations.
type ProcessService struct {
	client *Client
}

// Execute processes a transaction against a rule or ruleset and returns the result.
//
// Either RuleID or RulesetID is required. To evaluate a new transaction, set
// Transaction with a valid Type. To re-evaluate a transaction Aurora already
// stored, set TransactionID; Transaction may then be omitted, because Aurora
// evaluates the stored data rather than the request body.
func (s *ProcessService) Execute(ctx context.Context, req *ProcessRequest) (*ProcessResponse, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	httpReq, err := s.client.newRequest(ctx, http.MethodPost, pathProcess, req)
	if err != nil {
		return nil, err
	}

	var resp ProcessResponse
	if err := s.client.do(httpReq, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (req *ProcessRequest) validate() error {
	if req == nil {
		return &ValidationError{Field: "request", Message: "request is required"}
	}
	if req.RuleID == "" && req.RulesetID == "" {
		return &ValidationError{Field: "rule_id", Message: "either rule_id or ruleset_id is required"}
	}

	if req.TransactionID == "" {
		if req.Transaction == nil {
			return &ValidationError{Field: "transaction", Message: "transaction is required when transaction_id is empty"}
		}
		if !validTransactionTypes[req.Transaction.Type] {
			return &ValidationError{Field: "transaction.type", Message: "invalid or missing transaction type"}
		}
		return nil
	}

	// Re-evaluation: the type is optional, but reject one that is set and invalid.
	if req.Transaction != nil && req.Transaction.Type != "" && !validTransactionTypes[req.Transaction.Type] {
		return &ValidationError{Field: "transaction.type", Message: "invalid transaction type"}
	}
	return nil
}
