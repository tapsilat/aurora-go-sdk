package aurora

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client calls the Aurora API. Create one with NewClient and reuse it; it is
// safe for concurrent use.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client

	// timeout is the value set by WithTimeout. NewClient applies it after all
	// options have run, so the order of options does not matter.
	timeout time.Duration

	// Callback reports payment outcomes.
	Callback *CallbackService
	// Process evaluates transactions against rules.
	Process *ProcessService
}

// NewClient returns a client that authenticates with apiKey, sent as a Bearer
// token. WithBaseURL is required: without it, every call returns
// ErrMissingBaseURL. Requests time out after 30 seconds unless WithTimeout or
// WithHTTPClient sets otherwise. NewClient does not contact the API, so an
// invalid key or URL surfaces on the first call.
func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{apiKey: apiKey}
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}

	switch {
	case c.httpClient == nil:
		timeout := defaultTimeout
		if c.timeout > 0 {
			timeout = c.timeout
		}
		c.httpClient = &http.Client{Timeout: timeout}
	case c.timeout > 0:
		copied := *c.httpClient
		copied.Timeout = c.timeout
		c.httpClient = &copied
	}

	c.Callback = &CallbackService{client: c}
	c.Process = &ProcessService{client: c}

	return c
}

// newRequest builds an authenticated JSON request for path, relative to
// baseURL + basePath. A nil body sends no body.
func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	if c.baseURL == "" {
		return nil, ErrMissingBaseURL
	}
	url := c.baseURL + basePath + path

	var buf io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("aurora: marshal request body: %w", err)
		}
		buf = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, buf)
	if err != nil {
		return nil, fmt.Errorf("aurora: create request: %w", err)
	}

	req.Header.Set("Authorization", authScheme+c.apiKey)
	req.Header.Set("Content-Type", contentTypeJSON)
	req.Header.Set("Accept", contentTypeJSON)

	return req, nil
}

// drainBody discards a bounded amount of the remaining body so the
// underlying TCP connection can be reused by the HTTP transport pool.
func drainBody(body io.ReadCloser) {
	_, _ = io.Copy(io.Discard, io.LimitReader(body, maxDrainSize))
	body.Close()
}

// do sends req and decodes a successful JSON response into v, unless v is nil.
// A status of 400 or higher returns an *APIError. At most maxResponseSize
// bytes of the body are read.
func (c *Client) do(req *http.Request, v any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("aurora: send request: %w", err)
	}
	defer drainBody(resp.Body)

	limited := io.LimitReader(resp.Body, maxResponseSize)
	respBody, err := io.ReadAll(limited)
	if err != nil {
		return fmt.Errorf("aurora: read response body: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return newAPIError(resp.StatusCode, respBody)
	}

	if v != nil {
		if err := json.Unmarshal(respBody, v); err != nil {
			return fmt.Errorf("aurora: decode response: %w", err)
		}
	}

	return nil
}
