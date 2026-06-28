package recording

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestStoreSnapshotActiveSpecAndServeEvents(t *testing.T) {
	store := NewStore()
	if _, ok := store.ActiveSpec(); ok {
		t.Fatal("ActiveSpec() ok = true before start")
	}
	if err := store.Start(Spec{TargetBaseURL: "http://upstream.local", CaptureHeaders: map[string]json.RawMessage{"X-Trace": json.RawMessage(`{}`)}, RequestBodyPattern: "equalToJson", Persist: true}); err != nil {
		t.Fatalf("Start(): %v", err)
	}
	if spec, ok := store.ActiveSpec(); !ok || spec.TargetBaseURL != "http://upstream.local" {
		t.Fatalf("ActiveSpec() = %+v, %v", spec, ok)
	}

	requestHeaders := http.Header{"X-Trace": {"abc"}}
	responseHeaders := http.Header{"Content-Type": {"application/json"}, "Transfer-Encoding": {"chunked"}}
	store.AddServeEvent(ServeEvent{
		Method:          http.MethodPost,
		URL:             "/api/items?debug=true",
		Path:            "/api/items",
		RawQuery:        "debug=true",
		RequestHeaders:  requestHeaders,
		RequestBody:     []byte(`{"id":1}`),
		ResponseStatus:  http.StatusCreated,
		ResponseHeaders: responseHeaders,
		ResponseBody:    []byte(`{"ok":true}`),
		Source:          SourceStub,
	})
	requestHeaders.Set("X-Trace", "mutated")
	responseHeaders.Set("Content-Type", "text/plain")

	snapshot, err := store.Snapshot(Spec{CaptureHeaders: map[string]json.RawMessage{"X-Trace": json.RawMessage(`{}`)}, RequestBodyPattern: "equalToJson", Persist: true})
	if err != nil {
		t.Fatalf("Snapshot(): %v", err)
	}
	if snapshot.Meta.Total != 1 || len(snapshot.Mappings) != 1 {
		t.Fatalf("snapshot = %+v", snapshot)
	}

	var raw map[string]any
	if err := json.Unmarshal(mustMappingJSON(t, snapshot.Mappings[0]), &raw); err != nil {
		t.Fatalf("decode mapping: %v", err)
	}
	if raw["persistent"] != true {
		t.Fatalf("persistent = %v", raw["persistent"])
	}
	request := raw["request"].(map[string]any)
	if request["url"] != "/api/items?debug=true" {
		t.Fatalf("request url = %v", request["url"])
	}
	headers := request["headers"].(map[string]any)
	if headers["X-Trace"].(map[string]any)["equalTo"] != "abc" {
		t.Fatalf("captured header = %v", headers)
	}
	patterns := request["bodyPatterns"].([]any)
	if patterns[0].(map[string]any)["equalToJson"] != `{"id":1}` {
		t.Fatalf("bodyPatterns = %v", patterns)
	}
	response := raw["response"].(map[string]any)
	if _, ok := response["headers"].(map[string]any)["Transfer-Encoding"]; ok {
		t.Fatalf("excluded response header recorded: %v", response["headers"])
	}
	if response["jsonBody"] == nil {
		t.Fatalf("jsonBody missing: %v", response)
	}
}

func TestBuildSnapshotTextBinaryAndNameBranches(t *testing.T) {
	snapshot, err := BuildSnapshot([]ServeEvent{
		{
			Method:         http.MethodGet,
			Path:           "/text",
			ResponseStatus: http.StatusOK,
			ResponseHeaders: http.Header{
				"Content-Type": {"text/plain; charset=utf-8"},
			},
			ResponseBody: []byte("hello"),
		},
		{
			Method:         http.MethodGet,
			Path:           "/empty",
			ResponseStatus: http.StatusNoContent,
		},
		{
			Method:         http.MethodGet,
			URL:            "/url-only",
			Path:           "/url-only",
			ResponseStatus: http.StatusOK,
			ResponseHeaders: http.Header{
				"Content-Type": {"application/octet-stream"},
			},
			ResponseBody: []byte{0, 255},
		},
	}, Spec{RequestBodyPattern: "unsupported"})
	if err != nil {
		t.Fatalf("BuildSnapshot(): %v", err)
	}
	if snapshot.Meta.Total != 3 {
		t.Fatalf("total = %d", snapshot.Meta.Total)
	}
}

func TestValidateTargetBaseURLRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"http://[::1", "relative/path", "http://"} {
		if err := validateTargetBaseURL(value); err == nil {
			t.Fatalf("validateTargetBaseURL(%q) error = nil", value)
		}
	}
}

func TestHeaderValueCaseInsensitive(t *testing.T) {
	headers := http.Header{"x-trace": {"abc"}}
	if got := headerValue(headers, "X-Trace"); got != "abc" {
		t.Fatalf("headerValue = %q, want abc", got)
	}
	if got := headerValue(nil, "X-Trace"); got != "" {
		t.Fatalf("headerValue nil = %q, want empty", got)
	}
}

func TestCloneHelpers(t *testing.T) {
	if cloneHeaders(nil) != nil || cloneBytes(nil) != nil {
		t.Fatal("nil clone helper returned non-nil")
	}
	headers := http.Header{"X": {"1"}}
	clonedHeaders := cloneHeaders(headers)
	headers.Set("X", "2")
	if clonedHeaders.Get("X") != "1" {
		t.Fatalf("cloned header mutated: %v", clonedHeaders)
	}
	body := []byte("abc")
	clonedBody := cloneBytes(body)
	body[0] = 'x'
	if string(clonedBody) != "abc" {
		t.Fatalf("cloned body mutated: %q", clonedBody)
	}
}

func mustMappingJSON(t *testing.T, value any) []byte {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal(): %v", err)
	}
	return body
}
