package aurora

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// ErrMissingBaseURL is returned by every call on a client created without
// WithBaseURL. No request is sent.
var ErrMissingBaseURL = errors.New("aurora: base URL is not configured (use WithBaseURL)")

// Sentinels for the machine-readable error codes Aurora sends. Match them with
// errors.Is; the match uses APIError.Code, not the HTTP status.
var (
	// ErrTransactionNotFound: no transaction matches the given ID.
	ErrTransactionNotFound = &codeError{code: ErrorCodeTransactionNotFound}
	// ErrAmbiguousTransaction: a payment ID matches more than one transaction
	// and Aurora cannot pick one. APIError.CandidateCount says how many.
	ErrAmbiguousTransaction = &codeError{code: ErrorCodeAmbiguousTransaction}
	// ErrExternalIDConflict: the external ID is already bound to another
	// transaction.
	ErrExternalIDConflict = &codeError{code: ErrorCodeExternalIDConflict}
)

// codeError is the type of the error code sentinels.
type codeError struct{ code string }

func (e *codeError) Error() string { return "aurora: " + e.code }

// APIError is returned when Aurora responds with status 400 or higher. Use
// errors.As to get it, errors.Is with the ErrTransactionNotFound style
// sentinels, or the IsNotFound, IsUnauthorized and IsRateLimited helpers.
type APIError struct {
	// StatusCode is the HTTP status of the response.
	StatusCode int
	// Message is the error text from the response body. If the body is not
	// an Aurora error object, it holds the body itself, cut to 512 bytes, or
	// the standard status text when the body is empty.
	Message string
	// Body is the raw response body, for logging and debugging.
	Body []byte
	// Code is the "error_code" field of the body, such as
	// ErrorCodeTransactionNotFound. Empty when the endpoint does not send one.
	Code string
	// CandidateCount is the number of matching transactions, sent with
	// ErrorCodeAmbiguousTransaction. Zero otherwise.
	CandidateCount int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("aurora: api error (status %d): %s", e.StatusCode, e.Message)
}

// Is reports whether target is the sentinel for e.Code.
func (e *APIError) Is(target error) bool {
	t, ok := target.(*codeError)
	return ok && e.Code != "" && t.code == e.Code
}

// newAPIError builds an APIError from a failed response. See APIError.Message
// for how the message is chosen.
func newAPIError(statusCode int, body []byte) *APIError {
	apiErr := &APIError{
		StatusCode: statusCode,
		Message:    errorMessage(statusCode, body),
		Body:       body,
	}
	apiErr.Code, apiErr.CandidateCount = errorDetails(body)
	return apiErr
}

// errorDetails reads error_code and candidate_count from an error body. Each
// is read on its own, so a malformed field does not hide the other.
func errorDetails(body []byte) (code string, candidates int) {
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil {
		return "", 0
	}
	if raw, ok := fields["error_code"]; ok {
		_ = json.Unmarshal(raw, &code)
	}
	if raw, ok := fields["candidate_count"]; ok {
		_ = json.Unmarshal(raw, &candidates)
	}
	return code, candidates
}

func errorMessage(statusCode int, body []byte) string {
	var parsed ErrorResponse
	if err := json.Unmarshal(body, &parsed); err == nil {
		if parsed.Error != "" {
			return parsed.Error
		}
		if parsed.Message != "" {
			return parsed.Message
		}
	}

	text := strings.TrimSpace(string(body))
	if text == "" {
		return http.StatusText(statusCode)
	}
	if len(text) > maxErrorMessageLen {
		// ToValidUTF8 drops a multi-byte character cut in half.
		text = strings.ToValidUTF8(text[:maxErrorMessageLen], "") + "..."
	}
	return text
}

// ValidationError is returned when a request fails the SDK's checks. The
// request is not sent.
type ValidationError struct {
	// Field is the JSON name of the invalid field, such as "payment_id" or
	// "transaction.type", or "request" when the request itself is nil.
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("aurora: validation error: %s: %s", e.Field, e.Message)
}

// IsNotFound reports whether err is an API error with status 404. The Callback
// and Process endpoints report an unknown transaction ID or payment ID with
// status 500, so those cases do not match. The Outcomes endpoint returns 404
// with code transaction_not_found; errors.Is(err, ErrTransactionNotFound) is
// the code-based check for that.
func IsNotFound(err error) bool { return hasStatus(err, http.StatusNotFound) }

// IsUnauthorized reports whether err is an API error with status 401 or 403.
func IsUnauthorized(err error) bool {
	return hasStatus(err, http.StatusUnauthorized) || hasStatus(err, http.StatusForbidden)
}

// IsRateLimited reports whether err is an API error with status 429.
func IsRateLimited(err error) bool { return hasStatus(err, http.StatusTooManyRequests) }

func hasStatus(err error, status int) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == status
}
