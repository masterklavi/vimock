package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestLoggingMiddlewareIncludesRequestBody(t *testing.T) {
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logOutput, nil))
	handler := loggingMiddleware(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if string(body) != `{"id":"req-1"}` {
			t.Fatalf("handler body = %q, want request body", body)
		}
		w.WriteHeader(http.StatusAccepted)
	}))

	req := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(`{"id":"req-1"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	logRecord := decodeLogRecord(t, logOutput.Bytes())
	if logRecord["request_body"] != `{"id":"req-1"}` {
		t.Fatalf("logged request_body = %v, want request body", logRecord["request_body"])
	}
	if logRecord["status"] != float64(http.StatusAccepted) {
		t.Fatalf("logged status = %v, want %d", logRecord["status"], http.StatusAccepted)
	}
}

func TestLoggingMiddlewareSamplesUnreadRequestBody(t *testing.T) {
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logOutput, nil))
	handler := loggingMiddleware(logger, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest(http.MethodPost, "/unmatched", strings.NewReader(`{"method":"rests.get"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	logRecord := decodeLogRecord(t, logOutput.Bytes())
	if logRecord["request_body"] != `{"method":"rests.get"}` {
		t.Fatalf("logged request_body = %v, want unread request body", logRecord["request_body"])
	}
}

func TestLoggingMiddlewareDoesNotReadClosedRequestBody(t *testing.T) {
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logOutput, nil))
	handler := loggingMiddleware(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if string(body) != `{"id":"closed"}` {
			t.Fatalf("handler body = %q, want request body", body)
		}
		if err := r.Body.Close(); err != nil {
			t.Fatalf("close request body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/__admin/mappings", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = &errorAfterCloseReadCloser{reader: strings.NewReader(`{"id":"closed"}`)}
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	logRecord := decodeLogRecord(t, logOutput.Bytes())
	if logRecord["request_body"] != `{"id":"closed"}` {
		t.Fatalf("logged request_body = %v, want request body", logRecord["request_body"])
	}
	if _, ok := logRecord["request_body_read_error"]; ok {
		t.Fatalf("unexpected request_body_read_error = %v", logRecord["request_body_read_error"])
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

func decodeLogRecord(t *testing.T, data []byte) map[string]any {
	t.Helper()

	var record map[string]any
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatalf("decode log record: %v\n%s", err, data)
	}
	return record
}

type errorAfterCloseReadCloser struct {
	reader *strings.Reader
	closed bool
}

var errReadAfterClose = errors.New("invalid read on closed body")

func (r *errorAfterCloseReadCloser) Read(p []byte) (int, error) {
	if r.closed {
		return 0, errReadAfterClose
	}
	return r.reader.Read(p)
}

func (r *errorAfterCloseReadCloser) Close() error {
	r.closed = true
	return nil
}
