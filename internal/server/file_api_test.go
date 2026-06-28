package server

import (
	"bytes"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"vimock/internal/files"
	"vimock/internal/mapping"
)

func TestLegacyFileUploadWorkflow(t *testing.T) {
	fileStore := files.NewMemoryStore()
	handler := NewHandlerWithStores(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		mapping.NewStore(),
		fileStore,
	)

	login := requestWithBody(t, handler, http.MethodPost, "/api/login", "")
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d", login.Code, http.StatusOK)
	}
	token := login.Body.String()
	if token == "" {
		t.Fatalf("login token is empty")
	}

	fileName := "mc_product.dsc"
	payload := []byte{0x0a, 0x12, 0x76, 0x69, 0x6d, 0x6f, 0x63, 0x6b}
	createUpload := requestWithHeadersAndReader(
		t,
		handler,
		http.MethodPost,
		"/api/tus/"+fileName+"?override=true",
		map[string]string{
			"Tus-Resumable":   "1.0.0",
			"Upload-Length":   "8",
			"Upload-Metadata": "filename " + hex.EncodeToString([]byte(fileName)),
			"X-Auth":          token,
		},
		nil,
	)
	if createUpload.Code != http.StatusCreated {
		t.Fatalf("create upload status = %d, want %d: %s", createUpload.Code, http.StatusCreated, createUpload.Body.String())
	}
	if got := createUpload.Header().Get("Location"); got != "/api/tus/"+fileName {
		t.Fatalf("location = %q, want /api/tus/%s", got, fileName)
	}

	patchUpload := requestWithHeadersAndReader(
		t,
		handler,
		http.MethodPatch,
		"/api/tus/"+fileName+"?override=true",
		map[string]string{
			"Content-Type":    "application/offset+octet-stream",
			"Tus-Resumable":   "1.0.0",
			"Upload-Offset":   "0",
			"Upload-Metadata": "ignored",
			"X-Auth":          token,
		},
		bytes.NewReader(payload),
	)
	if patchUpload.Code != http.StatusNoContent {
		t.Fatalf("patch upload status = %d, want %d: %s", patchUpload.Code, http.StatusNoContent, patchUpload.Body.String())
	}
	if got := patchUpload.Header().Get("Upload-Offset"); got != "8" {
		t.Fatalf("patch upload offset = %q, want 8", got)
	}

	stored, ok := fileStore.Get(fileName)
	if !ok {
		t.Fatalf("uploaded file was not stored")
	}
	if !bytes.Equal(stored, payload) {
		t.Fatalf("stored bytes = %v, want %v", stored, payload)
	}

	createMapping(t, handler, `{
	  "request": {
	    "method": "GET",
	    "urlPath": "/descriptor"
	  },
	  "response": {
	    "status": 200,
	    "headers": {
	      "Content-Type": "application/octet-stream"
	    },
	    "bodyFileName": "mc_product.dsc"
	  }
	}`)
	bodyFile := requestWithBody(t, handler, http.MethodGet, "/descriptor", "")
	if bodyFile.Code != http.StatusOK {
		t.Fatalf("body file status = %d, want %d", bodyFile.Code, http.StatusOK)
	}
	if !bytes.Equal(bodyFile.Body.Bytes(), payload) {
		t.Fatalf("body file bytes = %v, want %v", bodyFile.Body.Bytes(), payload)
	}

	reset := requestWithBody(t, handler, http.MethodPost, "/__admin/ext/grpc/reset", "")
	if reset.Code != http.StatusOK {
		t.Fatalf("grpc reset status = %d, want %d", reset.Code, http.StatusOK)
	}
	if reset.Body.Len() != 0 {
		t.Fatalf("grpc reset body = %q, want empty", reset.Body.String())
	}
}

func TestLegacyFileUploadRejectsInvalidAuth(t *testing.T) {
	handler := newTestHandler()

	resp := requestWithHeadersAndReader(
		t,
		handler,
		http.MethodPost,
		"/api/tus/payload.bin?override=true",
		map[string]string{
			"Tus-Resumable":   "1.0.0",
			"Upload-Length":   "1",
			"Upload-Metadata": "filename " + hex.EncodeToString([]byte("payload.bin")),
			"X-Auth":          "wrong",
		},
		nil,
	)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusUnauthorized)
	}
}

func TestLegacyFileUploadValidationErrors(t *testing.T) {
	handler := newTestHandler()
	token := legacyFileAuthToken

	tests := []struct {
		name       string
		method     string
		path       string
		headers    map[string]string
		body       io.Reader
		wantStatus int
	}{
		{
			name:   "create requires override",
			method: http.MethodPost,
			path:   "/api/tus/payload.bin",
			headers: map[string]string{
				"Tus-Resumable":   "1.0.0",
				"Upload-Length":   "1",
				"Upload-Metadata": "filename " + hex.EncodeToString([]byte("payload.bin")),
				"X-Auth":          token,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "create requires tus version",
			method: http.MethodPost,
			path:   "/api/tus/payload.bin?override=true",
			headers: map[string]string{
				"Upload-Length":   "1",
				"Upload-Metadata": "filename " + hex.EncodeToString([]byte("payload.bin")),
				"X-Auth":          token,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "create rejects invalid upload length",
			method: http.MethodPost,
			path:   "/api/tus/payload.bin?override=true",
			headers: map[string]string{
				"Tus-Resumable":   "1.0.0",
				"Upload-Length":   "not-int",
				"Upload-Metadata": "filename " + hex.EncodeToString([]byte("payload.bin")),
				"X-Auth":          token,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "create rejects invalid hex filename metadata",
			method: http.MethodPost,
			path:   "/api/tus/payload.bin?override=true",
			headers: map[string]string{
				"Tus-Resumable":   "1.0.0",
				"Upload-Length":   "1",
				"Upload-Metadata": "filename not-hex",
				"X-Auth":          token,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "create rejects unsafe filename metadata",
			method: http.MethodPost,
			path:   "/api/tus/payload.bin?override=true",
			headers: map[string]string{
				"Tus-Resumable":   "1.0.0",
				"Upload-Length":   "1",
				"Upload-Metadata": "filename " + hex.EncodeToString([]byte("dir/payload.bin")),
				"X-Auth":          token,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "patch supports only zero offset",
			method: http.MethodPatch,
			path:   "/api/tus/payload.bin?override=true",
			headers: map[string]string{
				"Tus-Resumable": "1.0.0",
				"Upload-Offset": "1",
				"X-Auth":        token,
			},
			body:       bytes.NewReader([]byte{0x01}),
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := requestWithHeadersAndReader(t, handler, tt.method, tt.path, tt.headers, tt.body)
			if resp.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
		})
	}
}

func TestUploadMetadataFileName(t *testing.T) {
	name, ok, err := uploadMetadataFileName("owner ignored, filename " + hex.EncodeToString([]byte("payload.bin")))
	if err != nil {
		t.Fatalf("uploadMetadataFileName() error = %v", err)
	}
	if !ok || name != "payload.bin" {
		t.Fatalf("name = %q, ok = %v, want payload.bin true", name, ok)
	}

	name, ok, err = uploadMetadataFileName("owner ignored")
	if err != nil {
		t.Fatalf("uploadMetadataFileName() error = %v", err)
	}
	if ok || name != "" {
		t.Fatalf("name = %q, ok = %v, want empty false", name, ok)
	}
}

func TestValidateUploadFileName(t *testing.T) {
	valid, err := validateUploadFileName(" payload.bin ")
	if err != nil {
		t.Fatalf("validateUploadFileName() error = %v", err)
	}
	if valid != "payload.bin" {
		t.Fatalf("valid filename = %q, want payload.bin", valid)
	}

	for _, name := range []string{"", ".", "..", "dir/payload.bin", `dir\payload.bin`, "bad\x00name"} {
		t.Run(name, func(t *testing.T) {
			if _, err := validateUploadFileName(name); err == nil {
				t.Fatalf("validateUploadFileName(%q) error = nil, want error", name)
			}
		})
	}
}

func requestWithHeadersAndReader(t *testing.T, handler http.Handler, method, path string, headers map[string]string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, body)
	for name, value := range headers {
		req.Header.Set(name, value)
	}

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	return resp
}
