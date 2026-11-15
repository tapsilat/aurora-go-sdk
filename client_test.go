package aurora

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// recordedRequest captures what the test server received.
type recordedRequest struct {
	method string
	path   string
	header http.Header
	body   map[string]any
}

// newTestServer returns a client pointed at a server that records each
// request and replies with the given status and body.
func newTestServer(t *testing.T, status int, body string) (*Client, *recordedRequest, *atomic.Int32) {
	t.Helper()
	rec := &recordedRequest{}
	calls := &atomic.Int32{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		rec.method = r.Method
		rec.path = r.URL.Path
		rec.header = r.Header.Clone()
		raw, _ := io.ReadAll(r.Body)
		rec.body = nil
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &rec.body); err != nil {
				t.Errorf("request body is not JSON: %v: %s", err, raw)
			}
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return NewClient("test-key", WithBaseURL(srv.URL+"/")), rec, calls
}

func TestContextCancellation(t *testing.T) {
	client, _, _ := newTestServer(t, http.StatusOK, `{}`)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Process.Execute(ctx, &ProcessRequest{
		RuleID:      "r",
		Transaction: &Transaction{Type: TransactionTypeCard},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestNetworkErrorIsNotAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close() // nothing listens on url any more

	client := NewClient("key", WithBaseURL(url))
	_, err := client.Process.Execute(context.Background(), &ProcessRequest{
		RuleID:      "r",
		Transaction: &Transaction{Type: TransactionTypeCard},
	})
	if err == nil {
		t.Fatal("expected an error")
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		t.Errorf("network failure reported as APIError: %v", err)
	}
	if !strings.HasPrefix(err.Error(), "aurora: send request:") {
		t.Errorf("err = %v", err)
	}
}

func TestSuccessWithInvalidBody(t *testing.T) {
	for _, body := range []string{"<html>proxy</html>", ""} {
		client, _, _ := newTestServer(t, http.StatusOK, body)
		_, err := client.Process.Execute(context.Background(), &ProcessRequest{
			RuleID:      "r",
			Transaction: &Transaction{Type: TransactionTypeCard},
		})
		if err == nil || !strings.HasPrefix(err.Error(), "aurora: decode response:") {
			t.Errorf("body %q: err = %v, want decode error", body, err)
		}
	}
}

func TestInvalidBaseURL(t *testing.T) {
	client := NewClient("key", WithBaseURL("://no-scheme"))
	_, err := client.Process.Execute(context.Background(), &ProcessRequest{
		RuleID:      "r",
		Transaction: &Transaction{Type: TransactionTypeCard},
	})
	if err == nil || !strings.HasPrefix(err.Error(), "aurora: create request:") {
		t.Fatalf("err = %v, want create request error", err)
	}
}

func validProcessRequest() *ProcessRequest {
	return &ProcessRequest{RuleID: "r", Transaction: &Transaction{Type: TransactionTypeCard}}
}

// slowServer answers only after delay, or when the client gives up.
func slowServer(t *testing.T, delay time.Duration) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(delay):
			_, _ = io.WriteString(w, `{"allow": true}`)
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestClientTimeout(t *testing.T) {
	url := slowServer(t, 2*time.Second)
	client := NewClient("key", WithBaseURL(url), WithTimeout(50*time.Millisecond))

	start := time.Now()
	_, err := client.Process.Execute(context.Background(), validProcessRequest())
	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("err = %v, want a timeout", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("timeout took %v", elapsed)
	}
}

func TestContextDeadline(t *testing.T) {
	url := slowServer(t, 2*time.Second)
	client := NewClient("key", WithBaseURL(url)) // default 30s timeout is longer than the deadline

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := client.Process.Execute(ctx, validProcessRequest())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
}

// One client shared by many goroutines must not mix up requests and
// responses. Run with -race.
func TestConcurrentUse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ProcessRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(ProcessResponse{Allow: true, TransactionID: req.Transaction.Common.PaymentID})
	}))
	defer srv.Close()
	client := NewClient("key", WithBaseURL(srv.URL))

	const workers = 50
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := fmt.Sprintf("pay-%d", i)
			resp, err := client.Process.Execute(context.Background(), &ProcessRequest{
				RuleID:      "r",
				Transaction: &Transaction{Type: TransactionTypeCard, Common: &Common{PaymentID: id}},
			})
			switch {
			case err != nil:
				errs <- err
			case resp.TransactionID != id:
				errs <- fmt.Errorf("request %s got response for %s", id, resp.TransactionID)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

func TestOversizedResponse(t *testing.T) {
	big := strings.Repeat("x", maxResponseSize+1024)

	t.Run("success body is cut and fails to decode", func(t *testing.T) {
		client, _, _ := newTestServer(t, http.StatusOK, `{"allow_message":"`+big+`"}`)
		_, err := client.Process.Execute(context.Background(), validProcessRequest())
		if err == nil || !strings.HasPrefix(err.Error(), "aurora: decode response:") {
			t.Fatalf("err = %v, want decode error", err)
		}
	})

	t.Run("error body is capped", func(t *testing.T) {
		client, _, _ := newTestServer(t, http.StatusBadGateway, big)
		_, err := client.Process.Execute(context.Background(), validProcessRequest())
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err = %v, want APIError", err)
		}
		if len(apiErr.Body) != maxResponseSize {
			t.Errorf("Body length = %d, want %d", len(apiErr.Body), maxResponseSize)
		}
		if len(apiErr.Message) != maxErrorMessageLen+len("...") {
			t.Errorf("Message length = %d", len(apiErr.Message))
		}
	})
}

func TestBaseURLWithPathPrefix(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = io.WriteString(w, `{"allow": true}`)
	}))
	defer srv.Close()

	client := NewClient("key", WithBaseURL(srv.URL+"/aurora/"))
	if _, err := client.Process.Execute(context.Background(), validProcessRequest()); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/aurora/api/v1/process" {
		t.Errorf("path = %q, want /aurora/api/v1/process", gotPath)
	}
}

// 307 and 308 redirects keep the method, body and Authorization header on
// the same host. (301 and 302 turn a POST into a body-less GET in net/http;
// that case needs a code change, not a test that locks it in.)
func TestTemporaryRedirectKeepsRequest(t *testing.T) {
	for _, status := range []int{http.StatusTemporaryRedirect, http.StatusPermanentRedirect} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var method, auth string
			var body map[string]any
			mux := http.NewServeMux()
			mux.HandleFunc("/old/api/v1/process", func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/new/api/v1/process", status)
			})
			mux.HandleFunc("/new/api/v1/process", func(w http.ResponseWriter, r *http.Request) {
				method, auth = r.Method, r.Header.Get("Authorization")
				_ = json.NewDecoder(r.Body).Decode(&body)
				_, _ = io.WriteString(w, `{"allow": true}`)
			})
			srv := httptest.NewServer(mux)
			defer srv.Close()

			client := NewClient("key", WithBaseURL(srv.URL+"/old"))
			resp, err := client.Process.Execute(context.Background(), validProcessRequest())
			if err != nil {
				t.Fatal(err)
			}
			if !resp.Allow || method != http.MethodPost || auth != "Bearer key" || body["rule_id"] != "r" {
				t.Errorf("method=%s auth=%q body=%v", method, auth, body)
			}
		})
	}
}

// An API key that is not a valid header value fails before sending, and the
// error never contains the key.
func TestInvalidAPIKeyIsNotLeaked(t *testing.T) {
	client, _, calls := newTestServer(t, http.StatusOK, `{}`)
	secret := "sk_live_secret_value"
	client.apiKey = secret + "\n"

	_, err := client.Process.Execute(context.Background(), validProcessRequest())
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("error leaks the API key: %v", err)
	}
	if calls.Load() != 0 {
		t.Errorf("request with an invalid key reached the server")
	}
}

func TestEmptyAPIKeyIsRejectedByServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer")) == "" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"code":401,"message":"Unauthorized"}`)
			return
		}
		_, _ = io.WriteString(w, `{"allow": true}`)
	}))
	defer srv.Close()

	_, err := NewClient("", WithBaseURL(srv.URL)).Process.Execute(context.Background(), validProcessRequest())
	if !IsUnauthorized(err) {
		t.Fatalf("err = %v, want unauthorized", err)
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.Message != "Unauthorized" {
		t.Errorf("Message = %q", apiErr.Message)
	}
}
