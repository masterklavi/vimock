package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vimock/internal/mapping"
)

func TestMappingsCRUD(t *testing.T) {
	handler := newTestHandler()

	createResp := requestWithBody(t, handler, http.MethodPost, "/__admin/mappings", `{
	  "name": "Fry Proxy",
	  "persistent": true,
	  "request": {
	    "urlPattern": "/druz-fry/.*",
	    "method": "ANY"
	  },
	  "response": {
	    "status": 200,
	    "proxyBaseUrl": "http://fry-intgrtest-11.vseinstrumenti.net",
	    "proxyUrlPrefixToRemove": "/druz-fry"
	  },
	  "priority": 10,
	  "metadata": {
	    "wiremock-gui": {
	      "folder": "/DRUZ-Fry"
	    }
	  },
	  "x-extra": {
	    "kept": true
	  }
	}`)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", createResp.Code, http.StatusCreated, createResp.Body.String())
	}

	created := decodeObjectResponse(t, createResp)
	id, _ := created["id"].(string)
	if !mapping.IsValidID(id) {
		t.Fatalf("created id = %q, want UUID", id)
	}
	if created["name"] != "Fry Proxy" {
		t.Fatalf("created name = %v, want Fry Proxy", created["name"])
	}
	if created["persistent"] != true {
		t.Fatalf("persistent = %v, want true", created["persistent"])
	}
	if _, ok := created["x-extra"]; !ok {
		t.Fatalf("unknown field x-extra was not preserved")
	}

	listResp := requestWithBody(t, handler, http.MethodGet, "/__admin/mappings", "")
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listResp.Code, http.StatusOK)
	}
	list := decodeObjectResponse(t, listResp)
	mappings := list["mappings"].([]any)
	if len(mappings) != 1 {
		t.Fatalf("mappings len = %d, want 1", len(mappings))
	}
	meta := list["meta"].(map[string]any)
	if meta["total"] != float64(1) {
		t.Fatalf("meta.total = %v, want 1", meta["total"])
	}
	listed := mappings[0].(map[string]any)
	if listed["id"] != id {
		t.Fatalf("listed id = %v, want %s", listed["id"], id)
	}
	folder := listed["metadata"].(map[string]any)["wiremock-gui"].(map[string]any)["folder"]
	if folder != "/DRUZ-Fry" {
		t.Fatalf("folder = %v, want /DRUZ-Fry", folder)
	}

	getResp := requestWithBody(t, handler, http.MethodGet, "/__admin/mappings/"+id, "")
	if getResp.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getResp.Code, http.StatusOK)
	}

	updateResp := requestWithBody(t, handler, http.MethodPut, "/__admin/mappings/"+id, `{
	  "id": "22222222-2222-4222-8222-222222222222",
	  "name": "Fry Proxy Updated",
	  "request": {
	    "urlPattern": "/druz-fry/.*",
	    "method": "ANY"
	  },
	  "response": {
	    "status": 201
	  },
	  "metadata": {
	    "wiremock-gui": {
	      "folder": "/DRUZ-Fry"
	    }
	  }
	}`)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d: %s", updateResp.Code, http.StatusOK, updateResp.Body.String())
	}
	updated := decodeObjectResponse(t, updateResp)
	if updated["id"] != id {
		t.Fatalf("updated id = %v, want path id %s", updated["id"], id)
	}
	if updated["name"] != "Fry Proxy Updated" {
		t.Fatalf("updated name = %v, want Fry Proxy Updated", updated["name"])
	}

	deleteResp := requestWithBody(t, handler, http.MethodDelete, "/__admin/mappings/"+id, "")
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want %d", deleteResp.Code, http.StatusOK)
	}
	if strings.TrimSpace(deleteResp.Body.String()) != "{}" {
		t.Fatalf("delete body = %q, want {}", deleteResp.Body.String())
	}

	deleteAgainResp := requestWithBody(t, handler, http.MethodDelete, "/__admin/mappings/"+id, "")
	if deleteAgainResp.Code != http.StatusNotFound {
		t.Fatalf("delete again status = %d, want %d", deleteAgainResp.Code, http.StatusNotFound)
	}
}

func TestMappingsValidationErrors(t *testing.T) {
	handler := newTestHandler()

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantError  string
	}{
		{
			name:       "create invalid JSON",
			method:     http.MethodPost,
			path:       "/__admin/mappings",
			body:       `{`,
			wantStatus: http.StatusBadRequest,
			wantError:  "valid JSON object",
		},
		{
			name:       "create missing request",
			method:     http.MethodPost,
			path:       "/__admin/mappings",
			body:       `{"response":{"status":200}}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "request is required",
		},
		{
			name:       "get invalid id",
			method:     http.MethodGet,
			path:       "/__admin/mappings/not-a-uuid",
			wantStatus: http.StatusBadRequest,
			wantError:  "not-a-uuid is not a valid UUID",
		},
		{
			name:       "update missing id",
			method:     http.MethodPut,
			path:       "/__admin/mappings/11111111-1111-4111-8111-111111111111",
			body:       `{}`,
			wantStatus: http.StatusNotFound,
			wantError:  "No stub mapping found",
		},
		{
			name:       "delete invalid id",
			method:     http.MethodDelete,
			path:       "/__admin/mappings/not-a-uuid",
			wantStatus: http.StatusBadRequest,
			wantError:  "not-a-uuid is not a valid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := requestWithBody(t, handler, tt.method, tt.path, tt.body)
			if resp.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
			body := decodeObjectResponse(t, resp)
			errorsValue := body["errors"].([]any)
			first := errorsValue[0].(map[string]any)
			title := first["title"].(string)
			if !strings.Contains(title, tt.wantError) {
				t.Fatalf("error title = %q, want containing %q", title, tt.wantError)
			}
		})
	}
}

func newTestHandler() http.Handler {
	return NewHandlerWithStore(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		mapping.NewStore(),
	)
}

func requestWithBody(t *testing.T, handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	return resp
}

func decodeObjectResponse(t *testing.T, resp *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body %q: %v", resp.Body.String(), err)
	}
	return body
}
