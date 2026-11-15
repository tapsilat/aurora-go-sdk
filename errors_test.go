package aurora

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestAPIErrorParsing(t *testing.T) {
	long := strings.Repeat("x", maxErrorMessageLen+100)
	tests := []struct {
		name    string
		status  int
		body    string
		message string
	}{
		{"error key", http.StatusBadRequest, `{"error": "bad rule"}`, "bad rule"},
		{"message key", http.StatusBadRequest, `{"message": "status is required"}`, "status is required"},
		{"plain text", http.StatusBadGateway, "upstream down", "upstream down"},
		{"empty body", http.StatusServiceUnavailable, "", http.StatusText(http.StatusServiceUnavailable)},
		{"long body", http.StatusInternalServerError, long, long[:maxErrorMessageLen] + "..."},
		{"auth middleware shape", http.StatusUnauthorized, `{"code":401,"message":"Unauthorized"}`, "Unauthorized"},
		{"error wins over message", http.StatusBadRequest, `{"error":"bad rule","message":"ignored"}`, "bad rule"},
		{"error is an object", http.StatusBadRequest, `{"error":{"code":1}}`, `{"error":{"code":1}}`},
		{"error is a number", http.StatusBadRequest, `{"error":123}`, `{"error":123}`},
		{"empty error falls back to message", http.StatusBadRequest, `{"error":"","message":"status is required"}`, "status is required"},
		{"json null", http.StatusBadRequest, `null`, "null"},
		{"json array", http.StatusBadRequest, `[]`, "[]"},
		{"whitespace only", http.StatusServiceUnavailable, "  \n ", http.StatusText(http.StatusServiceUnavailable)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _, _ := newTestServer(t, tt.status, tt.body)
			_, err := client.Process.Execute(context.Background(), &ProcessRequest{
				RuleID:      "r",
				Transaction: &Transaction{Type: TransactionTypeCard},
			})
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("err = %v, want APIError", err)
			}
			if apiErr.StatusCode != tt.status || apiErr.Message != tt.message {
				t.Errorf("got (%d, %q), want (%d, %q)", apiErr.StatusCode, apiErr.Message, tt.status, tt.message)
			}
			if string(apiErr.Body) != tt.body {
				t.Errorf("Body = %q, want %q", apiErr.Body, tt.body)
			}
		})
	}
}

func TestErrorHelpers(t *testing.T) {
	wrap := func(status int) error { return errors.Join(errors.New("ctx"), &APIError{StatusCode: status}) }

	if !IsRateLimited(wrap(http.StatusTooManyRequests)) || IsRateLimited(wrap(http.StatusBadRequest)) {
		t.Error("IsRateLimited")
	}
	if !IsNotFound(wrap(http.StatusNotFound)) || IsNotFound(errors.New("plain")) {
		t.Error("IsNotFound")
	}
	if !IsUnauthorized(wrap(http.StatusUnauthorized)) || !IsUnauthorized(wrap(http.StatusForbidden)) || IsUnauthorized(nil) {
		t.Error("IsUnauthorized")
	}
}

func TestMissingBaseURL(t *testing.T) {
	client := NewClient("key")
	_, err := client.Process.Execute(context.Background(), &ProcessRequest{
		RuleID:      "r",
		Transaction: &Transaction{Type: TransactionTypeCard},
	})
	if !errors.Is(err, ErrMissingBaseURL) {
		t.Fatalf("err = %v, want ErrMissingBaseURL", err)
	}
}

// Callers log these strings; keep their format stable.
func TestErrorStrings(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{&APIError{StatusCode: 400, Message: "bad rule"}, "aurora: api error (status 400): bad rule"},
		{&ValidationError{Field: "payment_id", Message: "payment_id is required"}, "aurora: validation error: payment_id: payment_id is required"},
	}
	for _, tt := range tests {
		if got := tt.err.Error(); got != tt.want {
			t.Errorf("Error() = %q, want %q", got, tt.want)
		}
	}
}

func TestErrorMessageTruncationKeepsValidUTF8(t *testing.T) {
	// "ş" is two bytes; place one across the cut point.
	body := strings.Repeat("a", maxErrorMessageLen-1) + "ş" + "tail"
	msg := errorMessage(http.StatusBadGateway, []byte(body))
	if !utf8.ValidString(msg) {
		t.Fatalf("message is not valid UTF-8: %q", msg)
	}
	if !strings.HasSuffix(msg, "...") {
		t.Errorf("message not marked as truncated: %q", msg)
	}
}
