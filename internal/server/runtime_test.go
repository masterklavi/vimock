package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vimock/internal/files"
	"vimock/internal/mapping"
	"vimock/internal/proxy"
	"vimock/internal/response"
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

func TestRuntimeMatchesBodyQueryAndHeaders(t *testing.T) {
	handler := newTestHandler()
	createMapping(t, handler, `{
	  "priority": 1,
	  "request": {
	    "method": "POST",
	    "urlPath": "/matchers",
	    "queryParameters": {
	      "date": {
	        "equalTo": "2025-10-14"
	      }
	    },
	    "headers": {
	      "Content-Type": {
	        "equalTo": "application/json"
	      }
	    },
	    "bodyPatterns": [
	      {
	        "matchesJsonPath": "$.params.providers[?(@ == 'provider-1')]"
	      },
	      {
	        "matchesJsonPath": {
	          "expression": "$.params.missing",
	          "absent": true
	        }
	      }
	    ]
	  },
	  "response": {
	    "status": 200,
	    "body": "matched"
	  }
	}`)

	resp := requestWithHeadersAndBody(
		t,
		handler,
		http.MethodPost,
		"/matchers?date=2025-10-14",
		map[string]string{"Content-Type": "application/json"},
		`{"params":{"providers":["provider-1"]}}`,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if got := resp.Body.String(); got != "matched" {
		t.Fatalf("body = %q, want matched", got)
	}

	wrongBody := requestWithHeadersAndBody(
		t,
		handler,
		http.MethodPost,
		"/matchers?date=2025-10-14",
		map[string]string{"Content-Type": "application/json"},
		`{"params":{"providers":["other"]}}`,
	)
	if wrongBody.Code != http.StatusNotFound {
		t.Fatalf("wrong body status = %d, want %d", wrongBody.Code, http.StatusNotFound)
	}

	wrongQuery := requestWithHeadersAndBody(
		t,
		handler,
		http.MethodPost,
		"/matchers?date=2025-10-15",
		map[string]string{"Content-Type": "application/json"},
		`{"params":{"providers":["provider-1"]}}`,
	)
	if wrongQuery.Code != http.StatusNotFound {
		t.Fatalf("wrong query status = %d, want %d", wrongQuery.Code, http.StatusNotFound)
	}
}

func TestRuntimeMatchesEqualToJSON(t *testing.T) {
	handler := newTestHandler()
	createMapping(t, handler, `{
	  "request": {
	    "method": "POST",
	    "urlPath": "/equal-json",
	    "bodyPatterns": [
	      {
	        "equalToJson": "{\"a\":1,\"b\":2}"
	      }
	    ]
	  },
	  "response": {
	    "status": 200,
	    "body": "equal"
	  }
	}`)

	resp := requestWithBody(t, handler, http.MethodPost, "/equal-json", `{"b":2,"a":1}`)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}
	if got := resp.Body.String(); got != "equal" {
		t.Fatalf("body = %q, want equal", got)
	}
}

func TestRuntimeAppliesResponseTemplate(t *testing.T) {
	handler := newTestHandler()
	createMapping(t, handler, `{
	  "request": {
	    "method": "POST",
	    "urlPath": "/template"
	  },
	  "response": {
	    "status": 200,
	    "jsonBody": {
	      "requestId": "{{jsonPath request.body '$.requestId'}}"
	    },
	    "transformers": [
	      "response-template"
	    ]
	  }
	}`)

	resp := requestWithBody(t, handler, http.MethodPost, "/template", `{"requestId":"req-123"}`)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["requestId"] != "req-123" {
		t.Fatalf("requestId = %q, want req-123", body["requestId"])
	}
}

func TestRuntimeServesBodyFileBytes(t *testing.T) {
	fileStore := files.NewMemoryStore()
	want := []byte{0x00, 0x01, 0xff, 0x50, 0x44, 0x46}
	fileStore.Put("payload.bin", want)

	handler := NewHandlerWithStores(nil, mapping.NewStore(), fileStore)
	createMapping(t, handler, `{
	  "request": {
	    "method": "GET",
	    "urlPath": "/file"
	  },
	  "response": {
	    "status": 200,
	    "headers": {
	      "Content-Type": "application/octet-stream"
	    },
	    "bodyFileName": "payload.bin"
	  }
	}`)

	resp := requestWithBody(t, handler, http.MethodGet, "/file", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}
	if got := resp.Header().Get("Content-Type"); got != "application/octet-stream" {
		t.Fatalf("content-type = %q, want application/octet-stream", got)
	}
	if !bytes.Equal(resp.Body.Bytes(), want) {
		t.Fatalf("body bytes = %v, want %v", resp.Body.Bytes(), want)
	}
}

func TestRuntimeProxiesFallbackAfterPrioritySelection(t *testing.T) {
	var upstreamRequests int
	store := mapping.NewStore()
	createStoreMapping(t, store, `{
	  "priority": 10,
	  "request": {
	    "method": "ANY",
	    "urlPattern": "/proxy/.*"
	  },
	  "response": {
	    "proxyBaseUrl": "http://upstream.local",
	    "proxyUrlPrefixToRemove": "/proxy"
	  }
	}`)
	createStoreMapping(t, store, `{
	  "priority": 1,
	  "request": {
	    "method": "POST",
	    "urlPath": "/proxy/local"
	  },
	  "response": {
	    "status": 200,
	    "body": "local"
	  }
	}`)

	runtime := runtimeAPI{
		mappings: store,
		renderer: response.NewRenderer(nil),
		forwarder: proxy.NewForwarder(&http.Client{
			Transport: serverRoundTripFunc(func(r *http.Request) (*http.Response, error) {
				upstreamRequests++
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("read upstream body: %v", err)
				}
				return &http.Response{
					StatusCode: http.StatusAccepted,
					Header: http.Header{
						"X-Upstream": []string{"ok"},
					},
					Body: io.NopCloser(strings.NewReader(r.Method + " " + r.URL.RequestURI() + " " + string(body) + " " + r.Header.Get("X-Smoke"))),
				}, nil
			}),
		}),
	}

	local := requestRuntimeWithBody(t, runtime, http.MethodPost, "/proxy/local", "ignored")
	if local.Code != http.StatusOK {
		t.Fatalf("local status = %d, want %d", local.Code, http.StatusOK)
	}
	if local.Body.String() != "local" {
		t.Fatalf("local body = %q, want local", local.Body.String())
	}
	if upstreamRequests != 0 {
		t.Fatalf("upstreamRequests = %d, want 0 for higher priority local stub", upstreamRequests)
	}

	proxied := requestRuntimeWithHeadersAndBody(
		t,
		runtime,
		http.MethodPost,
		"/proxy/upstream?debug=true",
		map[string]string{"X-Smoke": "yes"},
		"payload",
	)
	if proxied.Code != http.StatusAccepted {
		t.Fatalf("proxied status = %d, want %d: %s", proxied.Code, http.StatusAccepted, proxied.Body.String())
	}
	if got := proxied.Header().Get("X-Upstream"); got != "ok" {
		t.Fatalf("X-Upstream = %q, want ok", got)
	}
	if got := proxied.Body.String(); got != "POST /upstream?debug=true payload yes" {
		t.Fatalf("proxied body = %q, want upstream echo", got)
	}
	if upstreamRequests != 1 {
		t.Fatalf("upstreamRequests = %d, want 1", upstreamRequests)
	}
}

type serverRoundTripFunc func(*http.Request) (*http.Response, error)

func (f serverRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestRuntimeAppliesFixedDelayAndChunkedDribble(t *testing.T) {
	store := mapping.NewStore()
	stub, err := mapping.ParseJSON([]byte(`{
	  "request": {
	    "method": "GET",
	    "urlPath": "/delayed"
	  },
	  "response": {
	    "status": 200,
	    "body": "abcdef",
	    "fixedDelayMilliseconds": 25,
	    "chunkedDribbleDelay": {
	      "numberOfChunks": 3,
	      "totalDuration": 30
	    }
	  }
	}`))
	if err != nil {
		t.Fatalf("parse mapping: %v", err)
	}
	store.Create(stub)

	var sleeps []time.Duration
	runtime := runtimeAPI{
		mappings: store,
		renderer: response.NewRenderer(nil),
		sleeper: func(_ context.Context, duration time.Duration) error {
			sleeps = append(sleeps, duration)
			return nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/delayed", nil)
	resp := httptest.NewRecorder()
	runtime.serveHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}
	if resp.Body.String() != "abcdef" {
		t.Fatalf("body = %q, want abcdef", resp.Body.String())
	}
	wantSleeps := []time.Duration{25 * time.Millisecond, 15 * time.Millisecond, 15 * time.Millisecond}
	if len(sleeps) != len(wantSleeps) {
		t.Fatalf("sleeps = %v, want %v", sleeps, wantSleeps)
	}
	for index, want := range wantSleeps {
		if sleeps[index] != want {
			t.Fatalf("sleeps[%d] = %s, want %s", index, sleeps[index], want)
		}
	}
}

func TestRuntimeSupportsStatefulScenarios(t *testing.T) {
	handler := newTestHandler()
	createMapping(t, handler, `{
	  "scenarioName": "job",
	  "requiredScenarioState": "Started",
	  "newScenarioState": "Running",
	  "request": {
	    "method": "GET",
	    "urlPath": "/job"
	  },
	  "response": {
	    "status": 202,
	    "body": "started"
	  },
	  "priority": 1
	}`)
	createMapping(t, handler, `{
	  "scenarioName": "job",
	  "requiredScenarioState": "Running",
	  "newScenarioState": "Done",
	  "request": {
	    "method": "GET",
	    "urlPath": "/job"
	  },
	  "response": {
	    "status": 202,
	    "body": "running"
	  },
	  "priority": 1
	}`)
	createMapping(t, handler, `{
	  "scenarioName": "job",
	  "requiredScenarioState": "Done",
	  "request": {
	    "method": "GET",
	    "urlPath": "/job"
	  },
	  "response": {
	    "status": 200,
	    "body": "done"
	  },
	  "priority": 1
	}`)

	first := requestWithBody(t, handler, http.MethodGet, "/job", "")
	if first.Code != http.StatusAccepted || first.Body.String() != "started" {
		t.Fatalf("first response = %d %q, want 202 started", first.Code, first.Body.String())
	}

	second := requestWithBody(t, handler, http.MethodGet, "/job", "")
	if second.Code != http.StatusAccepted || second.Body.String() != "running" {
		t.Fatalf("second response = %d %q, want 202 running", second.Code, second.Body.String())
	}

	third := requestWithBody(t, handler, http.MethodGet, "/job", "")
	if third.Code != http.StatusOK || third.Body.String() != "done" {
		t.Fatalf("third response = %d %q, want 200 done", third.Code, third.Body.String())
	}

	fourth := requestWithBody(t, handler, http.MethodGet, "/job", "")
	if fourth.Code != http.StatusOK || fourth.Body.String() != "done" {
		t.Fatalf("fourth response = %d %q, want 200 done", fourth.Code, fourth.Body.String())
	}
}

func TestRuntimeSkipsScenarioStubInWrongState(t *testing.T) {
	handler := newTestHandler()
	createMapping(t, handler, `{
	  "scenarioName": "choice",
	  "requiredScenarioState": "Later",
	  "request": {
	    "method": "GET",
	    "urlPath": "/choice"
	  },
	  "response": {
	    "body": "wrong-state"
	  },
	  "priority": 1
	}`)
	createMapping(t, handler, `{
	  "scenarioName": "choice",
	  "requiredScenarioState": "Started",
	  "request": {
	    "method": "GET",
	    "urlPath": "/choice"
	  },
	  "response": {
	    "body": "right-state"
	  },
	  "priority": 5
	}`)

	resp := requestWithBody(t, handler, http.MethodGet, "/choice", "")
	if resp.Code != http.StatusOK || resp.Body.String() != "right-state" {
		t.Fatalf("response = %d %q, want 200 right-state", resp.Code, resp.Body.String())
	}
}

func TestAdminResetsScenarioState(t *testing.T) {
	handler := newTestHandler()
	createMapping(t, handler, `{
	  "scenarioName": "resettable",
	  "requiredScenarioState": "Started",
	  "newScenarioState": "Second",
	  "request": {
	    "method": "GET",
	    "urlPath": "/resettable"
	  },
	  "response": {
	    "body": "first"
	  },
	  "priority": 1
	}`)
	createMapping(t, handler, `{
	  "scenarioName": "resettable",
	  "requiredScenarioState": "Second",
	  "request": {
	    "method": "GET",
	    "urlPath": "/resettable"
	  },
	  "response": {
	    "body": "second"
	  },
	  "priority": 1
	}`)

	first := requestWithBody(t, handler, http.MethodGet, "/resettable", "")
	if first.Body.String() != "first" {
		t.Fatalf("first body = %q, want first", first.Body.String())
	}
	second := requestWithBody(t, handler, http.MethodGet, "/resettable", "")
	if second.Body.String() != "second" {
		t.Fatalf("second body = %q, want second", second.Body.String())
	}

	reset := requestWithBody(t, handler, http.MethodPost, "/__admin/scenarios/reset", "")
	if reset.Code != http.StatusOK {
		t.Fatalf("reset status = %d, want %d", reset.Code, http.StatusOK)
	}
	if strings.TrimSpace(reset.Body.String()) != "{}" {
		t.Fatalf("reset body = %q, want {}", reset.Body.String())
	}

	afterReset := requestWithBody(t, handler, http.MethodGet, "/resettable", "")
	if afterReset.Body.String() != "first" {
		t.Fatalf("after reset body = %q, want first", afterReset.Body.String())
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

func createStoreMapping(t *testing.T, store *mapping.Store, body string) {
	t.Helper()

	stub, err := mapping.ParseJSON([]byte(body))
	if err != nil {
		t.Fatalf("parse mapping: %v", err)
	}
	store.Create(stub)
}

func requestRuntimeWithBody(t *testing.T, runtime runtimeAPI, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	return requestRuntimeWithHeadersAndBody(t, runtime, method, path, nil, body)
}

func requestRuntimeWithHeadersAndBody(t *testing.T, runtime runtimeAPI, method, path string, headers map[string]string, body string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(method, path, reader)
	for name, value := range headers {
		req.Header.Set(name, value)
	}

	resp := httptest.NewRecorder()
	runtime.serveHTTP(resp, req)
	return resp
}

func requestWithHeadersAndBody(t *testing.T, handler http.Handler, method, path string, headers map[string]string, body string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(method, path, reader)
	for name, value := range headers {
		req.Header.Set(name, value)
	}

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	return resp
}
