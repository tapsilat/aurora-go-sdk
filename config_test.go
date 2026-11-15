package aurora

import (
	"net/http"
	"testing"
	"time"
)

func TestClientOptions(t *testing.T) {
	t.Run("default timeout", func(t *testing.T) {
		c := NewClient("k", WithBaseURL("https://api.example.com"))
		if c.httpClient.Timeout != defaultTimeout {
			t.Errorf("Timeout = %v", c.httpClient.Timeout)
		}
	})

	t.Run("timeout does not modify the caller's client", func(t *testing.T) {
		own := &http.Client{Timeout: time.Minute}
		c := NewClient("k", WithHTTPClient(own), WithTimeout(5*time.Second))
		if own.Timeout != time.Minute {
			t.Errorf("caller's client changed: %v", own.Timeout)
		}
		if c.httpClient == own || c.httpClient.Timeout != 5*time.Second {
			t.Errorf("client timeout = %v", c.httpClient.Timeout)
		}
	})

	t.Run("option order does not matter", func(t *testing.T) {
		own := &http.Client{}
		c := NewClient("k", WithTimeout(5*time.Second), WithHTTPClient(own))
		if c.httpClient.Timeout != 5*time.Second {
			t.Errorf("Timeout = %v", c.httpClient.Timeout)
		}
	})

	t.Run("custom client is used as-is without timeout", func(t *testing.T) {
		own := &http.Client{Timeout: time.Minute}
		if c := NewClient("k", WithHTTPClient(own)); c.httpClient != own {
			t.Error("custom client was replaced")
		}
	})

	t.Run("nil client and nil option are ignored", func(t *testing.T) {
		c := NewClient("k", WithHTTPClient(nil), nil, WithTimeout(-1))
		if c.httpClient == nil || c.httpClient.Timeout != defaultTimeout {
			t.Errorf("httpClient = %+v", c.httpClient)
		}
	})

	t.Run("base URL is trimmed", func(t *testing.T) {
		c := NewClient("k", WithBaseURL(" https://api.example.com/ "))
		if c.baseURL != "https://api.example.com" {
			t.Errorf("baseURL = %q", c.baseURL)
		}
	})
}
