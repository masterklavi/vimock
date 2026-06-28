package server

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vimock/internal/files"
	"vimock/internal/grpcdesc"
	"vimock/internal/mapping"
	"vimock/internal/proxy"
	"vimock/internal/recording"
	"vimock/internal/response"
	"vimock/internal/scenario"
)

func TestAdminAdditionalErrorPaths(t *testing.T) {
	handler := newTestHandler()
	validID := "11111111-1111-4111-8111-111111111111"
	tests := []struct {
		method string
		path   string
		body   string
		want   int
	}{
		{http.MethodGet, "/__admin/mappings/" + validID, "", http.StatusNotFound},
		{http.MethodPut, "/__admin/mappings/not-a-uuid", `{}`, http.StatusBadRequest},
		{http.MethodPost, "/__admin/recordings/start", ``, http.StatusBadRequest},
		{http.MethodPost, "/__admin/recordings/start", `{`, http.StatusBadRequest},
		{http.MethodPost, "/__admin/recordings/stop", ``, http.StatusBadRequest},
		{http.MethodPost, "/__admin/recordings/snapshot", `{`, http.StatusBadRequest},
		{http.MethodPut, "/__admin/ext/grpc/descriptors/bad.txt", `bad`, http.StatusBadRequest},
		{http.MethodPut, "/__admin/ext/grpc/descriptors/source.proto", string([]byte{0xff}), http.StatusBadRequest},
		{http.MethodPost, "/__admin/ext/grpc/reset", ``, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			resp := requestWithBody(t, handler, tt.method, tt.path, tt.body)
			if resp.Code != tt.want {
				t.Fatalf("status = %d, want %d body=%s", resp.Code, tt.want, resp.Body.String())
			}
		})
	}
}

func TestAdminAdditionalSuccessPaths(t *testing.T) {
	handler := newTestHandler()

	create := requestWithBody(t, handler, http.MethodPost, "/__admin/mappings", `{"request":{"method":"GET","url":"/replace"},"response":{"status":200}}`)
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", create.Code, create.Body.String())
	}
	created := decodeObjectResponse(t, create)
	id, ok := created["id"].(string)
	if !ok || id == "" {
		t.Fatalf("created id = %v", created["id"])
	}

	update := requestWithBody(t, handler, http.MethodPut, "/__admin/mappings/"+id, `{"request":{"method":"GET","url":"/replace"},"response":{"status":201}}`)
	if update.Code != http.StatusOK {
		t.Fatalf("update status = %d body=%s", update.Code, update.Body.String())
	}

	badUpdate := requestWithBody(t, handler, http.MethodPut, "/__admin/mappings/"+id, `{`)
	if badUpdate.Code != http.StatusBadRequest {
		t.Fatalf("bad update status = %d body=%s", badUpdate.Code, badUpdate.Body.String())
	}

	start := requestWithBody(t, handler, http.MethodPost, "/__admin/recordings/start", `{"targetBaseUrl":"http://example.test"}`)
	if start.Code != http.StatusOK {
		t.Fatalf("start recording status = %d body=%s", start.Code, start.Body.String())
	}
	snapshot := requestWithBody(t, handler, http.MethodPost, "/__admin/recordings/snapshot", ``)
	if snapshot.Code != http.StatusOK {
		t.Fatalf("snapshot status = %d body=%s", snapshot.Code, snapshot.Body.String())
	}
	stop := requestWithBody(t, handler, http.MethodPost, "/__admin/recordings/stop", ``)
	if stop.Code != http.StatusOK {
		t.Fatalf("stop status = %d body=%s", stop.Code, stop.Body.String())
	}

	putProto := requestWithBody(t, handler, http.MethodPut, "/__admin/ext/grpc/descriptors/source.proto", `syntax = "proto3";`)
	if putProto.Code != http.StatusCreated {
		t.Fatalf("put proto status = %d body=%s", putProto.Code, putProto.Body.String())
	}
	putProto = requestWithBody(t, handler, http.MethodPut, "/__admin/ext/grpc/descriptors/source.proto", `syntax = "proto3"; package test;`)
	if putProto.Code != http.StatusOK {
		t.Fatalf("replace proto status = %d body=%s", putProto.Code, putProto.Body.String())
	}
	listProto := requestWithBody(t, handler, http.MethodGet, "/__admin/ext/grpc/descriptors", ``)
	if listProto.Code != http.StatusOK {
		t.Fatalf("list proto status = %d body=%s", listProto.Code, listProto.Body.String())
	}
	deleteProto := requestWithBody(t, handler, http.MethodDelete, "/__admin/ext/grpc/descriptors/source.proto", ``)
	if deleteProto.Code != http.StatusOK {
		t.Fatalf("delete proto status = %d body=%s", deleteProto.Code, deleteProto.Body.String())
	}
}

func TestReadHelpersHandleBodyReadErrors(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Body = errReadCloser{}
	if _, err := readMapping(w, req, ""); err == nil || !strings.Contains(err.Error(), "read mapping body") {
		t.Fatalf("readMapping error = %v", err)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	req.Body = errReadCloser{}
	if _, err := readRecordingSpec(w, req); err == nil || !strings.Contains(err.Error(), "read recording body") {
		t.Fatalf("readRecordingSpec error = %v", err)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	req.Body = errReadCloser{}
	if _, err := readOptionalRecordingSpec(w, req); err == nil || !strings.Contains(err.Error(), "read recording body") {
		t.Fatalf("readOptionalRecordingSpec error = %v", err)
	}
}

func TestFileAPIAdditionalValidationBranches(t *testing.T) {
	api := fileAPI{files: files.NewMemoryStore(), descriptors: grpcdesc.NewStore()}
	tests := []struct {
		name string
		req  *http.Request
		want int
	}{
		{
			name: "unauthorized patch",
			req:  httptest.NewRequest(http.MethodPatch, "/api/tus/file.bin?override=true", strings.NewReader("data")),
			want: http.StatusUnauthorized,
		},
		{
			name: "missing override",
			req: func() *http.Request {
				req := httptest.NewRequest(http.MethodPatch, "/api/tus/file.bin", strings.NewReader("data"))
				req.Header.Set("X-Auth", legacyFileAuthToken)
				req.Header.Set("Tus-Resumable", "1.0.0")
				req.Header.Set("Upload-Offset", "0")
				return req
			}(),
			want: http.StatusBadRequest,
		},
		{
			name: "bad offset",
			req: func() *http.Request {
				req := httptest.NewRequest(http.MethodPatch, "/api/tus/file.bin?override=true", strings.NewReader("data"))
				req.Header.Set("X-Auth", legacyFileAuthToken)
				req.Header.Set("Tus-Resumable", "1.0.0")
				req.Header.Set("Upload-Offset", "1")
				return req
			}(),
			want: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.req.SetPathValue("file", "file.bin")
			resp := httptest.NewRecorder()
			api.patchUpload(resp, tt.req)
			if resp.Code != tt.want {
				t.Fatalf("status = %d, want %d", resp.Code, tt.want)
			}
		})
	}
}

func TestRuntimeHelperBranches(t *testing.T) {
	first := mustParseMappingJSON(t, `{"request":{"method":"GET","urlPath":"/"},"response":{"status":200},"priority":2}`)
	second := mustParseMappingJSON(t, `{"request":{"method":"GET","urlPath":"/"},"response":{"status":200},"priority":1}`)
	selected, ok := selectBestStub([]mapping.Mapping{first, second})
	if !ok || selected.Priority() != 1 {
		t.Fatalf("selectBestStub = %+v %v", selected, ok)
	}
	if _, ok := selectBestStub(nil); ok {
		t.Fatal("selectBestStub(nil) ok = true")
	}

	w := httptest.NewRecorder()
	writeResponseRenderError(w, errors.New("render"))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("render error status = %d", w.Code)
	}
	w = httptest.NewRecorder()
	writeProxyError(w, errors.New("proxy"))
	if w.Code != http.StatusBadGateway {
		t.Fatalf("proxy error status = %d", w.Code)
	}

	parts := splitBody([]byte("abc"), 5)
	if len(parts) != 3 {
		t.Fatalf("splitBody small chunks len = %d", len(parts))
	}
}

func TestRuntimeRecordingProxyReadError(t *testing.T) {
	recorder := recording.NewStore()
	if err := recorder.Start(recording.Spec{TargetBaseURL: "http://upstream.local"}); err != nil {
		t.Fatalf("Start(): %v", err)
	}
	api := runtimeAPI{
		mappings:  mapping.NewStore(),
		renderer:  response.NewRenderer(nil),
		scenarios: scenario.NewStore(),
		recorder:  recorder,
	}
	req := httptest.NewRequest(http.MethodPost, "/unmatched", nil)
	req.Body = errReadCloser{}
	resp := httptest.NewRecorder()
	if !api.tryServeRecordingProxy(resp, req, newRequestBodyCache(req)) {
		t.Fatal("tryServeRecordingProxy should handle active recording")
	}
	if resp.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", resp.Code)
	}
}

func TestRuntimeAdditionalHelperBranches(t *testing.T) {
	api := runtimeAPI{}
	if api.tryServeRecordingProxy(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil), newRequestBodyCache(httptest.NewRequest(http.MethodGet, "/", nil))) {
		t.Fatal("tryServeRecordingProxy without recorder = true")
	}
	api.recorder = recording.NewStore()
	if api.tryServeRecordingProxy(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil), newRequestBodyCache(httptest.NewRequest(http.MethodGet, "/", nil))) {
		t.Fatal("tryServeRecordingProxy inactive recorder = true")
	}

	if err := writeResponseBody(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder(), nil, mapping.ResponseDefinition{}, nil); err != nil {
		t.Fatalf("writeResponseBody empty: %v", err)
	}
	if err := writeResponseBody(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder(), []byte("x"), mapping.ResponseDefinition{}, nil); err != nil {
		t.Fatalf("writeResponseBody one byte: %v", err)
	}
	err := writeResponseBody(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder(), []byte("abcd"), mapping.ResponseDefinition{ChunkedDribbleDelay: &mapping.ChunkedDribbleDelay{NumberOfChunks: 2, TotalDurationMilliseconds: 1}}, func(context.Context, time.Duration) error {
		return errors.New("sleep")
	})
	if err == nil || err.Error() != "sleep" {
		t.Fatalf("writeResponseBody sleeper error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/read-error", nil)
	req.Body = errReadCloser{}
	resp := httptest.NewRecorder()
	runtimeAPI{mappings: mapping.NewStore(), renderer: response.NewRenderer(nil)}.serveHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("serveHTTP read-error status = %d", resp.Code)
	}

	bodyStore := mapping.NewStore()
	bodyStore.Create(mustParseMappingJSON(t, `{"request":{"method":"POST","urlPath":"/body","bodyPatterns":[{"matchesJsonPath":"$.id"}]},"response":{"status":200}}`))
	req = httptest.NewRequest(http.MethodPost, "/body", nil)
	req.Body = errReadCloser{}
	resp = httptest.NewRecorder()
	runtimeAPI{mappings: bodyStore, renderer: response.NewRenderer(nil)}.serveHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("serveHTTP body read-error status = %d", resp.Code)
	}

	proxyStore := mapping.NewStore()
	proxyStore.Create(mustParseMappingJSON(t, `{"request":{"method":"GET","urlPath":"/proxy"},"response":{"status":200,"proxyBaseUrl":"://bad"}}`))
	resp = httptest.NewRecorder()
	runtimeAPI{mappings: proxyStore, renderer: response.NewRenderer(nil), forwarder: proxy.NewForwarder(nil)}.serveHTTP(resp, httptest.NewRequest(http.MethodGet, "/proxy", nil))
	if resp.Code != http.StatusBadGateway {
		t.Fatalf("serveHTTP proxy error status = %d", resp.Code)
	}
}

func TestFileAPIPatchUploadReadError(t *testing.T) {
	api := fileAPI{files: files.NewMemoryStore(), descriptors: grpcdesc.NewStore()}
	req := httptest.NewRequest(http.MethodPatch, "/api/tus/file.bin?override=true", nil)
	req.SetPathValue("file", "file.bin")
	req.Header.Set("X-Auth", legacyFileAuthToken)
	req.Header.Set("Tus-Resumable", "1.0.0")
	req.Header.Set("Upload-Offset", "0")
	req.Body = errReadCloser{}
	resp := httptest.NewRecorder()
	api.patchUpload(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.Code)
	}
}

type errReadCloser struct{}

func (errReadCloser) Read([]byte) (int, error) { return 0, errors.New("read") }
func (errReadCloser) Close() error             { return nil }

var _ io.ReadCloser = errReadCloser{}

func TestRuntimeAdditionalResponseBranches(t *testing.T) {
	store := mapping.NewStore()
	store.Create(mustParseMappingJSON(t, `{"request":{"method":"GET","urlPath":"/missing-file"},"response":{"status":200,"bodyFileName":"missing.json"}}`))
	handler := NewHandlerWithStore(nil, store)
	resp := requestWithBody(t, handler, http.MethodGet, "/missing-file", "")
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("missing body file status = %d, want 500", resp.Code)
	}

	w := httptest.NewRecorder()
	writeProxyResponse(w, httptest.NewRequest(http.MethodGet, "/", nil), proxy.Response{Status: http.StatusOK, Headers: http.Header{"Content-Length": {"3"}}, Body: []byte("abc")}, mapping.ResponseDefinition{ChunkedDribbleDelay: &mapping.ChunkedDribbleDelay{NumberOfChunks: 2, TotalDurationMilliseconds: 1}}, func(context.Context, time.Duration) error {
		return nil
	})
	if w.Header().Get("Content-Length") != "" || w.Body.String() != "abc" {
		t.Fatalf("proxy chunked response headers=%v body=%q", w.Header(), w.Body.String())
	}

	if compareStubOrder(mustParseMappingJSON(t, `{"request":{"method":"GET","urlPath":"/"},"response":{"status":200},"priority":1}`), mustParseMappingJSON(t, `{"request":{"method":"GET","urlPath":"/"},"response":{"status":200},"priority":1}`)) != 0 {
		t.Fatal("compareStubOrder identical zero value mappings should be equal")
	}
}
