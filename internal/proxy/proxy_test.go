package proxy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"vimock/internal/mapping"
)

func TestProxyForwardingAndPrefixRemoval(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			body, err := json.Marshal(map[string]string{
				"method": r.Method,
				"path":   r.URL.RequestURI(),
				"body":   readBody(t, r),
				"header": r.Header.Get("X-Request"),
			})
			if err != nil {
				t.Fatalf("marshal response: %v", err)
			}
			return &http.Response{
				StatusCode: http.StatusAccepted,
				Header: http.Header{
					"X-Upstream": []string{"ok"},
				},
				Body: io.NopCloser(strings.NewReader(string(body))),
			}, nil
		}),
	}

	original := httptest.NewRequest(http.MethodPost, "/proxy/api/items?debug=true", strings.NewReader("payload"))
	original.Header.Set("X-Request", "kept")
	original.Header.Set("Connection", "close")

	response, err := NewForwarder(client).Forward(
		context.Background(),
		original,
		[]byte("payload"),
		mapping.ResponseDefinition{
			ProxyBaseURL:           "http://upstream.local",
			ProxyURLPrefixToRemove: "/proxy",
		},
	)
	if err != nil {
		t.Fatalf("Forward() error = %v", err)
	}
	if response.Status != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Status, http.StatusAccepted)
	}
	if got := response.Headers.Get("X-Upstream"); got != "ok" {
		t.Fatalf("X-Upstream = %q, want ok", got)
	}

	var body map[string]string
	if err := json.Unmarshal(response.Body, &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["method"] != http.MethodPost {
		t.Fatalf("method = %q, want POST", body["method"])
	}
	if body["path"] != "/api/items?debug=true" {
		t.Fatalf("path = %q, want /api/items?debug=true", body["path"])
	}
	if body["body"] != "payload" {
		t.Fatalf("body = %q, want payload", body["body"])
	}
	if body["header"] != "kept" {
		t.Fatalf("header = %q, want kept", body["header"])
	}
}

func TestTargetURLJoinsBasePathAndQuery(t *testing.T) {
	original, err := url.Parse("/prefix/v1/items?q=1")
	if err != nil {
		t.Fatalf("parse original URL: %v", err)
	}

	target, err := TargetURL("https://example.com/base?token=abc", "/prefix", original)
	if err != nil {
		t.Fatalf("TargetURL() error = %v", err)
	}
	if target != "https://example.com/base/v1/items?token=abc&q=1" {
		t.Fatalf("target = %q, want joined URL", target)
	}
}

func TestTargetURLRejectsInvalidBaseURL(t *testing.T) {
	original, err := url.Parse("/items")
	if err != nil {
		t.Fatalf("parse original URL: %v", err)
	}

	if _, err := TargetURL("://bad", "", original); err == nil {
		t.Fatalf("TargetURL() error = nil, want invalid base URL error")
	}
	if _, err := TargetURL("/relative", "", original); err == nil {
		t.Fatalf("TargetURL() error = nil, want missing scheme/host error")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func readBody(t *testing.T, r *http.Request) string {
	t.Helper()

	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	return string(data)
}
