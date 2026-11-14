package aurora

import (
	"net/http"
	"strings"
	"time"
)

const (
	basePath       = "/api/v1"
	defaultTimeout = 30 * time.Second
)

// API endpoints, relative to basePath.
const (
	pathProcess             = "/process"
	pathCallbackTransaction = "/callback/transaction"
)

// Request headers.
const (
	authScheme      = "Bearer "
	contentTypeJSON = "application/json"
)

const (
	maxResponseSize = 10 << 20 // 10 MB
	// maxErrorMessageLen bounds the message taken from a non-JSON error body,
	// such as an HTML page from a proxy.
	maxErrorMessageLen = 512
	// maxDrainSize bounds how much unread body is discarded to keep the
	// connection reusable. A larger remainder is not worth reading; closing
	// the body then simply drops the connection.
	maxDrainSize = 64 << 10 // 64 KB
)

// Option is a functional option for configuring the Client.
type Option func(*Client)

// WithBaseURL sets the host URL (e.g. "https://api.example.com").
// The API path (/api/v1) is appended automatically. It is required: requests
// made by a client without a base URL return ErrMissingBaseURL.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(strings.TrimSpace(url), "/")
	}
}

// WithHTTPClient sets a custom HTTP client. A nil client is ignored.
// The client is never modified; WithTimeout applies to a copy of it.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// WithTimeout sets the HTTP request timeout (default: 30s). Non-positive
// values are ignored. When combined with WithHTTPClient, the timeout is set
// on a copy, so the caller's client is left unchanged.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.timeout = d
		}
	}
}
