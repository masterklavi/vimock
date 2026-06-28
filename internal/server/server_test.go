package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	resp := request(t, http.MethodGet, "/__admin/health")

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}
	if got := resp.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want application/json", got)
	}

	var body statusResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Status != "healthy" || body.Service != "vimock" {
		t.Fatalf("body = %+v, want healthy vimock response", body)
	}
}

func TestReady(t *testing.T) {
	resp := request(t, http.MethodGet, "/__admin/ready")

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}

	var body statusResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Status != "ready" || body.Service != "vimock" {
		t.Fatalf("body = %+v, want ready vimock response", body)
	}
}

func TestUnsupportedRouteReturnsNotFound(t *testing.T) {
	resp := request(t, http.MethodGet, "/unknown")

	if resp.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusNotFound)
	}
}

func request(t *testing.T, method, path string) *httptest.ResponseRecorder {
	t.Helper()

	handler := NewHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))
	req := httptest.NewRequest(method, path, nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	return resp
}
