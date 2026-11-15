package aurora

import (
	"context"
	"net/http"
	"strings"
)

// CallbackService reports payment outcomes back to Aurora.
type CallbackService struct {
	client *Client
}

// Transaction records the final outcome of a payment (POST /callback/transaction).
// Aurora finds the transaction in your organization whose common.payment_id
// equals req.PaymentID and stores req.Status and req.Message on it. A success
// outcome is also added to the device history that rules can check.
//
// The outcome is final once recorded:
//   - Sending the same status again succeeds and changes nothing.
//   - Sending a different status for a transaction that already has one
//     returns an *APIError.
//   - A PaymentID that matches no transaction, or more than one, returns an
//     *APIError.
//
// Aurora currently reports both errors with status 500, not 404 or 409.
//
// A request that fails validation returns a *ValidationError without
// contacting Aurora.
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

// validate applies Aurora's own checks before sending, so a request Aurora
// would reject fails without a network call.
func (req *CallbackTransactionRequest) validate() error {
	if req == nil {
		return &ValidationError{Field: "request", Message: "request is required"}
	}
	// Aurora's input check accepts a whitespace-only payment_id, but such an ID
	// matches no transaction and Aurora then fails with 500.
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
