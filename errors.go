package aurora

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// ErrMissingBaseURL is returned when a request is made by a client that was
// created without WithBaseURL.
var ErrMissingBaseURL = errors.New("aurora: base URL is not configured (use WithBaseURL)")

// APIError represents an error returned by the Aurora API.
type APIError struct {
	StatusCode int
	Message    string
	// Body is the raw response body, kept for logging and debugging.
	Body []byte
}

func (e *APIError) Error() string {
	return fmt.Sprintf("aurora: api error (status %d): %s", e.StatusCode, e.Message)
}

// newAPIError builds an APIError from a failed response. Aurora returns
// either {"error": "..."} or {"message": "..."} depending on the endpoint;
// any other body is used as-is, truncated.
func newAPIError(statusCode int, body []byte) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Message:    errorMessage(statusCode, body),
		Body:       body,
	}
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

// ValidationError represents a client-side validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("aurora: validation error: %s: %s", e.Field, e.Message)
}

// IsNotFound reports whether err is an API error with status 404.
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
