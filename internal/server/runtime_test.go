package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestRuntimeServesBodyForURLMatch(t *testing.T) {
	handler := newTestHandler()
	createMapping(t, handler, `{
	  "request": {
	    "method": "GET",
	    "url": "/hello?name=vimock"
	  },
	  "response": {
	    "status": 201,
	    "headers": {
	      "X-Test": "ok"
	    },
	    "body": "hello from vimock"
	  }
	}`)

	resp := requestWithBody(t, handler, http.MethodGet, "/hello?name=vimock", "")
	if resp.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusCreated)
	}
	if got := resp.Header().Get("X-Test"); got != "ok" {
		t.Fatalf("X-Test = %q, want ok", got)
	}
	if got := resp.Body.String(); got != "hello from vimock" {
		t.Fatalf("body = %q, want hello from vimock", got)
	}

	unmatched := requestWithBody(t, handler, http.MethodGet, "/hello?name=other", "")
	if unmatched.Code != http.StatusNotFound {
		t.Fatalf("unmatched status = %d, want %d", unmatched.Code, http.StatusNotFound)
	}
}

func TestRuntimeServesJSONBodyForURLPathMatch(t *testing.T) {
	handler := newTestHandler()
	createMapping(t, handler, `{
	  "request": {
	    "method": "GET",
	    "urlPath": "/json"
	  },
	  "response": {
	    "status": 200,
	    "jsonBody": {
	      "ok": true
	    }
	  }
	}`)

	resp := requestWithBody(t, handler, http.MethodGet, "/json?debug=true", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}
	if got := resp.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want application/json", got)
	}

	var body map[string]bool
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if !body["ok"] {
		t.Fatalf("body ok = false, want true")
	}
}

func TestRuntimeServesURLPatternFullMatchAndANYMethod(t *testing.T) {
	handler := newTestHandler()
	createMapping(t, handler, `{
	  "request": {
	    "method": "ANY",
	    "urlPattern": "/items/[0-9]+"
	  },
	  "response": {
	    "status": 202,
	    "body": "pattern"
	  }
	}`)

	resp := requestWithBody(t, handler, http.MethodPost, "/items/123", "")
	if resp.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusAccepted)
	}
	if got := resp.Body.String(); got != "pattern" {
		t.Fatalf("body = %q, want pattern", got)
	}

	unmatched := requestWithBody(t, handler, http.MethodPost, "/prefix/items/123", "")
	if unmatched.Code != http.StatusNotFound {
		t.Fatalf("partial regex status = %d, want %d", unmatched.Code, http.StatusNotFound)
	}
}

func TestRuntimeSelectsLowestPriorityThenInsertionOrder(t *testing.T) {
	handler := newTestHandler()
	createMapping(t, handler, `{
	  "name": "fallback",
	  "priority": 10,
	  "request": {
	    "method": "ANY",
	    "urlPattern": "/priority/.*"
	  },
	  "response": {
	    "status": 200,
	    "body": "fallback"
	  }
	}`)
	createMapping(t, handler, `{
	  "name": "exact",
	  "priority": 1,
	  "request": {
	    "method": "GET",
	    "urlPath": "/priority/item"
	  },
	  "response": {
	    "status": 200,
	    "body": "exact"
	  }
	}`)
	createMapping(t, handler, `{
	  "name": "first-tie",
	  "priority": 3,
	  "request": {
	    "method": "GET",
	    "urlPath": "/tie"
	  },
	  "response": {
	    "status": 200,
	    "body": "first"
	  }
	}`)
	createMapping(t, handler, `{
	  "name": "second-tie",
	  "priority": 3,
	  "request": {
	    "method": "GET",
	    "urlPath": "/tie"
	  },
	  "response": {
	    "status": 200,
	    "body": "second"
	  }
	}`)

	exact := requestWithBody(t, handler, http.MethodGet, "/priority/item", "")
	if got := exact.Body.String(); got != "exact" {
		t.Fatalf("priority response = %q, want exact", got)
	}

	fallback := requestWithBody(t, handler, http.MethodGet, "/priority/other", "")
	if got := fallback.Body.String(); got != "fallback" {
		t.Fatalf("fallback response = %q, want fallback", got)
	}

	tie := requestWithBody(t, handler, http.MethodGet, "/tie", "")
	if got := tie.Body.String(); got != "first" {
		t.Fatalf("tie response = %q, want first", got)
	}
}

func TestRuntimeDeletedMappingStopsMatching(t *testing.T) {
	handler := newTestHandler()
	id := createMapping(t, handler, `{
	  "request": {
	    "method": "GET",
	    "urlPath": "/temporary"
	  },
	  "response": {
	    "status": 200,
	    "body": "active"
	  }
	}`)

	active := requestWithBody(t, handler, http.MethodGet, "/temporary", "")
	if active.Code != http.StatusOK {
		t.Fatalf("active status = %d, want %d", active.Code, http.StatusOK)
	}

	deleted := requestWithBody(t, handler, http.MethodDelete, "/__admin/mappings/"+id, "")
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want %d", deleted.Code, http.StatusOK)
	}

	inactive := requestWithBody(t, handler, http.MethodGet, "/temporary", "")
	if inactive.Code != http.StatusNotFound {
		t.Fatalf("inactive status = %d, want %d", inactive.Code, http.StatusNotFound)
	}
}

func TestRuntimeNoMappingsReturnsWireMockLikeNotFound(t *testing.T) {
	handler := newTestHandler()

	resp := requestWithBody(t, handler, http.MethodGet, "/missing", "")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusNotFound)
	}
	if got := resp.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Fatalf("content-type = %q, want text/plain", got)
	}
	if got := resp.Body.String(); got != noMappingsMessage {
		t.Fatalf("body = %q, want %q", got, noMappingsMessage)
	}
}

func createMapping(t *testing.T, handler http.Handler, body string) string {
	t.Helper()

	resp := requestWithBody(t, handler, http.MethodPost, "/__admin/mappings", body)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create mapping status = %d, want %d: %s", resp.Code, http.StatusCreated, resp.Body.String())
	}
	created := decodeObjectResponse(t, resp)
	id, ok := created["id"].(string)
	if !ok || id == "" {
		t.Fatalf("created id = %v, want non-empty string", created["id"])
	}
	return id
}
