package aurora

import (
	"context"
	"net/http"
)

// ProcessService evaluates transactions against rules.
type ProcessService struct {
	client *Client
}

// Execute evaluates a transaction against a rule or a ruleset and returns the
// decision (POST /process). Set RuleID or RulesetID; if both are set, Aurora
// uses RuleID.
//
// It works in one of two modes:
//   - New transaction: leave TransactionID empty and set Transaction with a
//     valid Type. Aurora stores the transaction before evaluating it, and
//     ProcessResponse.TransactionID holds its ID.
//   - Re-evaluation: set TransactionID to an ID returned earlier. Aurora
//     evaluates the stored data and ignores Transaction. If a reviewer has
//     approved the transaction's case, Aurora returns Allow without running
//     any rules.
//
// A rule that fails during evaluation does not produce an error from
// Execute; it sets ProcessResponse.Error instead.
//
// Execute returns:
//   - *ValidationError when the request is invalid. Nothing is sent.
//   - ErrMissingBaseURL when the client has no base URL. Nothing is sent.
//   - *APIError when Aurora responds with status 400 or higher. An unknown
//     TransactionID currently comes back as status 500, not 404.
//   - A wrapped network or decoding error otherwise.
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

// validate rejects requests Aurora cannot process. When both rule_id and
// ruleset_id are empty, Aurora fails only after storing the transaction, so
// catching that here also avoids leaving an orphan transaction behind.
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
		return validateAttributes(req.Transaction.Attributes)
	}

	// Re-evaluation: Aurora ignores Transaction, but a set, invalid type is
	// still rejected because it points to a mistake in the caller's code.
	if req.Transaction != nil && req.Transaction.Type != "" && !validTransactionTypes[req.Transaction.Type] {
		return &ValidationError{Field: "transaction.type", Message: "invalid transaction type"}
	}
	return nil
}
