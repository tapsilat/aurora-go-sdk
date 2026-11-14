package aurora

import (
	"context"
	"net/http"
	"strings"
)

// CallbackService handles callback operations.
type CallbackService struct {
	client *Client
}

// Transaction reports the final outcome of a payment that Aurora evaluated.
//
// PaymentID must match the common.payment_id sent with the process request.
// Status is CallbackStatusSuccess or CallbackStatusFailed, and Message is
// required. Sending the same status again is safe: Aurora accepts it without
// changing anything.
func (s *CallbackService) Transaction(ctx context.Context, req *CallbackTransactionRequest) (*CallbackTransactionResponse, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	httpReq, err := s.client.newRequest(ctx, http.MethodPost, pathCallbackTransaction, req)
	if err != nil {
		return nil, err
	}

	var resp CallbackTransactionResponse
	if err := s.client.do(httpReq, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// validate mirrors the checks Aurora applies, so a request the API would
// reject with 400 fails here without a round trip.
func (req *CallbackTransactionRequest) validate() error {
	if req == nil {
		return &ValidationError{Field: "request", Message: "request is required"}
	}
	// A blank payment_id passes Aurora's 400 check but can never match a
	// transaction, so Aurora rejects it later with 500.
	if strings.TrimSpace(req.PaymentID) == "" {
		return &ValidationError{Field: "payment_id", Message: "payment_id is required"}
	}
	if req.Status == "" {
		return &ValidationError{Field: "status", Message: "status is required"}
	}
	if req.Status != CallbackStatusSuccess && req.Status != CallbackStatusFailed {
		return &ValidationError{Field: "status", Message: "status must be success or failed"}
	}
	// Aurora accepts a whitespace-only message, so only an empty one is rejected.
	if req.Message == "" {
		return &ValidationError{Field: "message", Message: "message is required"}
	}
	return nil
}
