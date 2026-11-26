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
	pathIntegrationSamples  = "/integration/samples"
	pathIntegrationIdentity = "/integration/identity"
	pathIntegrationConnect  = "/integration/connection"
	pathOutcomes            = "/outcomes"
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

// Option configures a Client. Pass options to NewClient. A nil Option is
// ignored.
type Option func(*Client)

// WithBaseURL sets the API host, for example "https://api.example.com".
// Do not include /api/v1: the client adds it. Surrounding spaces and
// trailing slashes are removed. This option is required; without it, every
// call returns ErrMissingBaseURL.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(strings.TrimSpace(url), "/")
	}
}

// WithHTTPClient makes the client send requests through hc, for example to
// use a custom transport or proxy. Its own Timeout is kept unless WithTimeout
// is also given. hc is never modified, and a nil hc is ignored.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// WithTimeout sets the time limit for each call, covering the connection,
// the request and reading the response. The default is 30 seconds. A zero or
// negative d is ignored. With WithHTTPClient, the timeout is set on a copy of
// that client. A deadline on the call's context also applies, and whichever
// ends first wins.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.timeout = d
		}
	}
}
