package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"vimock/internal/files"
	"vimock/internal/mapping"
	"vimock/internal/recording"
	"vimock/internal/response"
	"vimock/internal/scenario"
)

func TestServeGRPCErrorBranches(t *testing.T) {
	api := runtimeAPI{}
	req := httptest.NewRequest(http.MethodPost, "/svc/Method", nil)
	req.Header.Set("Content-Type", "application/grpc")
	resp := httptest.NewRecorder()
	api.serveGRPC(resp, req)
	assertGRPCStatus(t, resp, grpcStatusUnimplemented, "descriptor registry is not configured")

	descriptorStore := newLoadedDescriptorStore(t)
	api = runtimeAPI{descriptors: descriptorStore, mappings: mapping.NewStore(), renderer: response.NewRenderer(files.NewMemoryStore()), scenarios: scenario.NewStore()}
	resp = httptest.NewRecorder()
	api.serveGRPC(resp, httptest.NewRequest(http.MethodPost, "/missing.Service/Call", nil))
	assertGRPCStatus(t, resp, grpcStatusUnimplemented, "No matching gRPC service method found")

	resp = httptest.NewRecorder()
	badFrameReq := httptest.NewRequest(http.MethodPost, pdmFullMethod, nil)
	badFrameReq.Header.Set("Content-Type", "application/grpc")
	api.serveGRPC(resp, badFrameReq)
	assertGRPCStatus(t, resp, grpcStatusInternal, "gRPC request frame is too short")

	resp = httptest.NewRecorder()
	badProtoReq := httptest.NewRequest(http.MethodPost, pdmFullMethod, bytesReader(encodeGRPCFrame([]byte("bad"))))
	badProtoReq.Header.Set("Content-Type", "application/grpc")
	api.serveGRPC(resp, badProtoReq)
	assertGRPCStatus(t, resp, grpcStatusInternal, "decode gRPC request protobuf")
}

func TestGRPCHelperAdditionalBranches(t *testing.T) {
	api := runtimeAPI{descriptors: newLoadedDescriptorStore(t)}
	if _, _, ok := api.findGRPCMethod("bad"); ok {
		t.Fatal("bad path found method")
	}
	if service, method, ok := splitGRPCPath("/service/method"); !ok || service != "service" || method != "method" {
		t.Fatalf("splitGRPCPath = %q %q %v", service, method, ok)
	}
	if _, _, ok := splitGRPCPath("/missing"); ok {
		t.Fatal("splitGRPCPath missing method ok = true")
	}
	registry := api.descriptors.Active()
	method := mustGRPCMethod(t, registry, pdmServiceName, pdmMethodName)
	if _, err := decodeProtoJSON([]byte("bad"), method.Input(), registry); err == nil {
		t.Fatal("decodeProtoJSON bad payload error = nil")
	}
	if _, err := encodeProtoJSON([]byte(`{"bad":`), method.Output(), registry); err == nil {
		t.Fatal("encodeProtoJSON bad JSON error = nil")
	}
}

func TestRecordGRPCServeEventBranches(t *testing.T) {
	api := runtimeAPI{}
	api.recordGRPCServeEvent(httptest.NewRequest(http.MethodPost, pdmFullMethod, nil), []byte(`{}`), http.StatusOK, nil, nil)

	recorder := recording.NewStore()
	api.recorder = recorder
	api.recordGRPCServeEvent(httptest.NewRequest(http.MethodPost, pdmFullMethod, nil), []byte(`{"id":1}`), http.StatusOK, http.Header{"X": {"Y"}}, []byte(`{"ok":true}`))
	snapshot, err := recorder.Snapshot(recording.Spec{})
	if err != nil {
		t.Fatalf("Snapshot(): %v", err)
	}
	if snapshot.Meta.Total != 1 {
		t.Fatalf("events total = %d", snapshot.Meta.Total)
	}
}

func bytesReader(data []byte) *bytes.Reader { return bytes.NewReader(data) }
