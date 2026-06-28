package server

import (
	"bytes"
	"encoding/hex"
	"net/http"
	"os"
	"testing"

	"vimock/internal/files"
	"vimock/internal/grpcdesc"
	"vimock/internal/mapping"
)

func TestGRPCDescriptorAdminAPI(t *testing.T) {
	descriptorSet := readDescriptorFixture(t)
	handler := NewHandlerWithStoresAndDescriptors(nil, mapping.NewStore(), files.NewMemoryStore(), grpcdesc.NewStore())

	created := requestWithHeadersAndReader(
		t,
		handler,
		http.MethodPut,
		"/__admin/ext/grpc/descriptors/mc_product.dsc",
		map[string]string{"Content-Type": "application/octet-stream"},
		bytes.NewReader(descriptorSet),
	)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", created.Code, http.StatusCreated, created.Body.String())
	}
	assertDescriptorList(t, decodeObjectResponse(t, created), 1, 0)

	list := requestWithBody(t, handler, http.MethodGet, "/__admin/ext/grpc/descriptors", "")
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d: %s", list.Code, http.StatusOK, list.Body.String())
	}
	assertDescriptorList(t, decodeObjectResponse(t, list), 1, 0)

	reset := requestWithBody(t, handler, http.MethodPost, "/__admin/ext/grpc/reset", "")
	if reset.Code != http.StatusOK {
		t.Fatalf("reset status = %d, want %d: %s", reset.Code, http.StatusOK, reset.Body.String())
	}
	if reset.Body.Len() != 0 {
		t.Fatalf("reset body = %q, want empty", reset.Body.String())
	}

	reloaded := requestWithBody(t, handler, http.MethodGet, "/__admin/ext/grpc/descriptors", "")
	assertDescriptorList(t, decodeObjectResponse(t, reloaded), 1, 1)

	replaced := requestWithHeadersAndReader(
		t,
		handler,
		http.MethodPut,
		"/__admin/ext/grpc/descriptors/mc_product.dsc",
		map[string]string{"Content-Type": "application/octet-stream"},
		bytes.NewReader(descriptorSet),
	)
	if replaced.Code != http.StatusOK {
		t.Fatalf("replace status = %d, want %d: %s", replaced.Code, http.StatusOK, replaced.Body.String())
	}

	deleted := requestWithBody(t, handler, http.MethodDelete, "/__admin/ext/grpc/descriptors/mc_product.dsc", "")
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want %d: %s", deleted.Code, http.StatusOK, deleted.Body.String())
	}
	if body := bytes.TrimSpace(deleted.Body.Bytes()); !bytes.Equal(body, []byte("{}")) {
		t.Fatalf("delete body = %q, want {}", body)
	}

	deletedAgain := requestWithBody(t, handler, http.MethodDelete, "/__admin/ext/grpc/descriptors/mc_product.dsc", "")
	if deletedAgain.Code != http.StatusNotFound {
		t.Fatalf("second delete status = %d, want %d", deletedAgain.Code, http.StatusNotFound)
	}

	resetAfterDelete := requestWithBody(t, handler, http.MethodPost, "/__admin/ext/grpc/reset", "")
	if resetAfterDelete.Code != http.StatusOK {
		t.Fatalf("reset after delete status = %d, want %d", resetAfterDelete.Code, http.StatusOK)
	}
	empty := requestWithBody(t, handler, http.MethodGet, "/__admin/ext/grpc/descriptors", "")
	assertDescriptorList(t, decodeObjectResponse(t, empty), 0, 0)
}

func TestGRPCDescriptorAdminAPIRejectsInvalidDescriptors(t *testing.T) {
	handler := newTestHandler()

	tests := []struct {
		name string
		path string
		body []byte
	}{
		{
			name: "invalid descriptor set",
			path: "/__admin/ext/grpc/descriptors/bad.dsc",
			body: []byte("not a descriptor set"),
		},
		{
			name: "unsupported extension",
			path: "/__admin/ext/grpc/descriptors/bad.txt",
			body: readDescriptorFixture(t),
		},
		{
			name: "empty proto source",
			path: "/__admin/ext/grpc/descriptors/service.proto",
			body: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := requestWithHeadersAndReader(
				t,
				handler,
				http.MethodPut,
				tt.path,
				map[string]string{"Content-Type": "application/octet-stream"},
				bytes.NewReader(tt.body),
			)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d: %s", resp.Code, http.StatusBadRequest, resp.Body.String())
			}
		})
	}
}

func TestLegacyDescriptorUploadFeedsGRPCRegistry(t *testing.T) {
	descriptorSet := readDescriptorFixture(t)
	handler := NewHandlerWithStoresAndDescriptors(nil, mapping.NewStore(), files.NewMemoryStore(), grpcdesc.NewStore())

	token := legacyFileAuthToken
	fileName := "mc_product.dsc"
	uploadPath := "grpc/" + fileName
	createUpload := requestWithHeadersAndReader(
		t,
		handler,
		http.MethodPost,
		"/api/tus/"+uploadPath+"?override=true",
		map[string]string{
			"Tus-Resumable":   "1.0.0",
			"Upload-Length":   "1",
			"Upload-Metadata": "filename " + hex.EncodeToString([]byte(fileName)),
			"X-Auth":          token,
		},
		nil,
	)
	if createUpload.Code != http.StatusCreated {
		t.Fatalf("create upload status = %d, want %d: %s", createUpload.Code, http.StatusCreated, createUpload.Body.String())
	}

	patchUpload := requestWithHeadersAndReader(
		t,
		handler,
		http.MethodPatch,
		"/api/tus/"+uploadPath+"?override=true",
		map[string]string{
			"Content-Type":  "application/offset+octet-stream",
			"Tus-Resumable": "1.0.0",
			"Upload-Offset": "0",
			"X-Auth":        token,
		},
		bytes.NewReader(descriptorSet),
	)
	if patchUpload.Code != http.StatusNoContent {
		t.Fatalf("patch upload status = %d, want %d: %s", patchUpload.Code, http.StatusNoContent, patchUpload.Body.String())
	}

	list := requestWithBody(t, handler, http.MethodGet, "/__admin/ext/grpc/descriptors", "")
	assertDescriptorList(t, decodeObjectResponse(t, list), 1, 0)

	reset := requestWithBody(t, handler, http.MethodPost, "/__admin/ext/grpc/reset", "")
	if reset.Code != http.StatusOK {
		t.Fatalf("reset status = %d, want %d: %s", reset.Code, http.StatusOK, reset.Body.String())
	}
	reloaded := requestWithBody(t, handler, http.MethodGet, "/__admin/ext/grpc/descriptors", "")
	assertDescriptorList(t, decodeObjectResponse(t, reloaded), 1, 1)
}

func readDescriptorFixture(t *testing.T) []byte {
	t.Helper()

	data, err := os.ReadFile("../../testdata/mc_product.dsc")
	if err != nil {
		t.Fatalf("read descriptor fixture: %v", err)
	}
	return data
}

func assertDescriptorList(t *testing.T, body map[string]any, wantTotal int, minActiveFiles int) {
	t.Helper()

	meta := body["meta"].(map[string]any)
	if meta["total"] != float64(wantTotal) {
		t.Fatalf("meta.total = %v, want %d", meta["total"], wantTotal)
	}
	descriptors := body["descriptors"].([]any)
	if len(descriptors) != wantTotal {
		t.Fatalf("descriptors len = %d, want %d", len(descriptors), wantTotal)
	}
	registry := body["registry"].(map[string]any)
	files := int(registry["files"].(float64))
	if files < minActiveFiles {
		t.Fatalf("registry.files = %d, want >= %d", files, minActiveFiles)
	}
}
