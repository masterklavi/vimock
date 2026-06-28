package server

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGRPCStatusMappingHelpers(t *testing.T) {
	statusTests := []struct {
		name string
		code int
	}{
		{"OK", grpcStatusOK},
		{"cancelled", grpcStatusCanceled},
		{"UNKNOWN", grpcStatusUnknown},
		{"INVALID-ARGUMENT", grpcStatusInvalidArgument},
		{"NOT_FOUND", grpcStatusNotFound},
		{"PERMISSION_DENIED", grpcStatusPermission},
		{"UNIMPLEMENTED", grpcStatusUnimplemented},
		{"INTERNAL", grpcStatusInternal},
		{"UNAVAILABLE", grpcStatusUnavailable},
		{"UNAUTHENTICATED", grpcStatusUnauthenticated},
	}
	for _, tt := range statusTests {
		code, ok := grpcStatusCodeByName(tt.name)
		if !ok || code != tt.code {
			t.Fatalf("grpcStatusCodeByName(%q) = %d %v, want %d true", tt.name, code, ok, tt.code)
		}
	}
	if _, ok := grpcStatusCodeByName("bad"); ok {
		t.Fatal("unknown status should not resolve")
	}

	code, reason, ok := grpcStatusFromHeaders(http.Header{"Grpc-Status-Name": {"NOT_FOUND"}, "Grpc-Status-Reason": {"missing"}})
	if !ok || code != grpcStatusNotFound || reason != "missing" {
		t.Fatalf("grpcStatusFromHeaders = %d %q %v", code, reason, ok)
	}
	code, reason, ok = grpcStatusFromHeaders(http.Header{"grpc-status-name": {"bad"}})
	if !ok || code != grpcStatusInternal || reason == "" {
		t.Fatalf("unsupported header status = %d %q %v", code, reason, ok)
	}
	if _, _, ok := grpcStatusFromHeaders(nil); ok {
		t.Fatal("empty headers should not contain status")
	}
}

func TestGRPCStatusFromHTTP(t *testing.T) {
	tests := []struct {
		status int
		code   int
		ok     bool
	}{
		{http.StatusOK, grpcStatusOK, false},
		{http.StatusBadRequest, grpcStatusInternal, true},
		{http.StatusUnauthorized, grpcStatusUnauthenticated, true},
		{http.StatusForbidden, grpcStatusPermission, true},
		{http.StatusNotFound, grpcStatusUnimplemented, true},
		{http.StatusTooManyRequests, grpcStatusUnavailable, true},
		{http.StatusInternalServerError, grpcStatusUnknown, true},
		{http.StatusCreated, grpcStatusOK, false},
	}
	for _, tt := range tests {
		code, _, ok := grpcStatusFromHTTP(tt.status)
		if code != tt.code || ok != tt.ok {
			t.Fatalf("grpcStatusFromHTTP(%d) = %d %v", tt.status, code, ok)
		}
	}
}

func TestGRPCHeadersAndMessageHelpers(t *testing.T) {
	w := httptest.NewRecorder()
	writeGRPCErrorWithHeaders(w, http.Header{
		"X-Visible":          {"yes"},
		"grpc-status-name":   {"INTERNAL"},
		"grpc-status-reason": {"hidden"},
		"Content-Length":     {"42"},
	}, grpcStatusInternal, "bad request")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if w.Header().Get("X-Visible") != "yes" || w.Header().Get("Content-Length") != "" {
		t.Fatalf("headers = %v", w.Header())
	}
	if w.Header().Get("Grpc-Status") != "13" || w.Header().Get("Grpc-Message") != "bad%20request" {
		t.Fatalf("trailers = %v", w.Header())
	}

	if !isInternalGRPCMappingHeader("grpc-status-name") || !isInternalGRPCMappingHeader("content-length") || isInternalGRPCMappingHeader("x-visible") {
		t.Fatal("internal header classification mismatch")
	}
	if headerValue(http.Header{"x-custom": {"one"}}, "X-Custom") != "one" || headerValue(nil, "X") != "" {
		t.Fatal("headerValue mismatch")
	}
}

func TestGRPCBinaryMetadataHelpers(t *testing.T) {
	headers := grpcMatcherHeaders(http.Header{
		"trace-bin": {base64.StdEncoding.EncodeToString([]byte{1, 255})},
		"raw-bin":   {base64.RawStdEncoding.EncodeToString([]byte{2, 3})},
		"bad-bin":   {"%%%"},
		"x-plain":   {"value"},
	})
	if headers["trace-bin"][0] != "[1, -1]" || headers["raw-bin"][0] != "[2, 3]" || headers["bad-bin"][0] != "%%%" || headers["x-plain"][0] != "value" {
		t.Fatalf("converted headers = %v", headers)
	}
	if got := formatByteArray(nil); got != "[]" {
		t.Fatalf("formatByteArray(nil) = %q", got)
	}
	if _, ok := decodeGRPCBinaryHeader("%%%"); ok {
		t.Fatal("invalid binary header decoded")
	}
}

func TestGRPCFrameDecodeErrors(t *testing.T) {
	if _, err := decodeUnaryGRPCFrame(nil); err == nil {
		t.Fatal("empty frame error = nil")
	}
	if _, err := decodeUnaryGRPCFrame([]byte{1, 0, 0, 0, 0}); err == nil {
		t.Fatal("compressed frame error = nil")
	}
	if _, err := decodeUnaryGRPCFrame([]byte{0, 0, 0, 0, 2, 1}); err == nil {
		t.Fatal("truncated frame error = nil")
	}
}
