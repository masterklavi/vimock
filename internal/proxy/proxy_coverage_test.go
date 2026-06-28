package proxy

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"vimock/internal/mapping"
)

func TestNewForwarderDefaultClientAndForwardErrors(t *testing.T) {
	if NewForwarder(nil).client == nil {
		t.Fatal("default client should be set")
	}
	forwarder := NewForwarder(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network")
	})})
	_, err := forwarder.Forward(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil), nil, mapping.ResponseDefinition{ProxyBaseURL: "http://upstream.local"})
	if err == nil || !strings.Contains(err.Error(), "proxy request") {
		t.Fatalf("Forward() error = %v", err)
	}
	_, err = forwarder.Forward(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil), nil, mapping.ResponseDefinition{ProxyBaseURL: "://bad"})
	if err == nil {
		t.Fatal("Forward invalid target error = nil")
	}
}

func TestForwardReadBodyError(t *testing.T) {
	forwarder := NewForwarder(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Connection": {"close"}}, Body: errReader{}}, nil
	})})
	_, err := forwarder.Forward(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil), nil, mapping.ResponseDefinition{ProxyBaseURL: "http://upstream.local"})
	if err == nil || !strings.Contains(err.Error(), "read proxy response") {
		t.Fatalf("Forward() error = %v", err)
	}
}

func TestJoinURLPathAndQueryBranches(t *testing.T) {
	tests := []struct{ base, request, want string }{
		{"", "/r", "/r"},
		{"/base", "", "/base"},
		{"/base/", "/r", "/base/r"},
		{"/base", "r", "/base/r"},
	}
	for _, tt := range tests {
		if got := joinURLPath(tt.base, tt.request); got != tt.want {
			t.Fatalf("joinURLPath(%q,%q)=%q want %q", tt.base, tt.request, got, tt.want)
		}
	}
	if got := joinRawQuery("", "q=1"); got != "q=1" {
		t.Fatalf("joinRawQuery request only = %q", got)
	}
	if got := joinRawQuery("token=1", ""); got != "token=1" {
		t.Fatalf("joinRawQuery base only = %q", got)
	}
	original, _ := url.Parse("/prefix")
	target, err := TargetURL("http://upstream.local/base", "/prefix", original)
	if err != nil || target != "http://upstream.local/base" {
		t.Fatalf("TargetURL prefix to empty = %q %v", target, err)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read") }
func (errReader) Close() error             { return nil }

var _ io.ReadCloser = errReader{}
