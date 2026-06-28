package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"vimock/internal/files"
	"vimock/internal/grpcdesc"
	"vimock/internal/mapping"
	"vimock/internal/proxy"
	"vimock/internal/recording"
)

func TestRecordingStartProxyStopActivatesMappings(t *testing.T) {
	upstreamCalls := 0
	client := &http.Client{
		Transport: recordingRoundTripFunc(func(r *http.Request) (*http.Response, error) {
			upstreamCalls++
			if r.URL.String() != "http://upstream.local/api/items?sku=1" {
				t.Fatalf("upstream URL = %q", r.URL.String())
			}
			if r.Header.Get("X-Request") != "kept" {
				t.Fatalf("X-Request = %q, want kept", r.Header.Get("X-Request"))
			}
			return &http.Response{
				StatusCode: http.StatusCreated,
				Header: http.Header{
					"Content-Type": []string{"application/json"},
					"X-Upstream":   []string{"ok"},
				},
				Body: io.NopCloser(strings.NewReader(`{"ok":true}`)),
			}, nil
		}),
	}

	mappings := mapping.NewStore()
	handler := NewHandlerWithStoresDescriptorsRecorderForwarder(
		nil,
		mappings,
		files.NewMemoryStore(),
		grpcdesc.NewStore(),
		recording.NewStore(),
		proxy.NewForwarder(client),
	)

	start := requestWithBody(t, handler, http.MethodPost, "/__admin/recordings/start", `{
	  "targetBaseUrl": "http://upstream.local",
	  "captureHeaders": {
	    "X-Request": {}
	  }
	}`)
	if start.Code != http.StatusOK {
		t.Fatalf("start status = %d, want 200: %s", start.Code, start.Body.String())
	}

	proxied := requestWithHeadersAndBody(
		t,
		handler,
		http.MethodGet,
		"/api/items?sku=1",
		map[string]string{"X-Request": "kept"},
		"",
	)
	if proxied.Code != http.StatusCreated {
		t.Fatalf("proxied status = %d, want 201: %s", proxied.Code, proxied.Body.String())
	}
	if proxied.Body.String() != `{"ok":true}` {
		t.Fatalf("proxied body = %q", proxied.Body.String())
	}

	stop := requestWithBody(t, handler, http.MethodPost, "/__admin/recordings/stop", "")
	if stop.Code != http.StatusOK {
		t.Fatalf("stop status = %d, want 200: %s", stop.Code, stop.Body.String())
	}
	assertSnapshotTotal(t, stop.Body.Bytes(), 1)
	if mappings.Count() != 1 {
		t.Fatalf("mappings count = %d, want 1", mappings.Count())
	}

	playback := requestWithHeadersAndBody(
		t,
		handler,
		http.MethodGet,
		"/api/items?sku=1",
		map[string]string{"X-Request": "kept"},
		"",
	)
	if playback.Code != http.StatusCreated {
		t.Fatalf("playback status = %d, want 201: %s", playback.Code, playback.Body.String())
	}
	if !strings.Contains(playback.Body.String(), `"ok":true`) {
		t.Fatalf("playback body = %q, want recorded JSON", playback.Body.String())
	}
	if upstreamCalls != 1 {
		t.Fatalf("upstream calls = %d, want 1", upstreamCalls)
	}
}

func TestRecordingSnapshotCreatesMappingsFromServeEvents(t *testing.T) {
	mappings := mapping.NewStore()
	handler := NewHandlerWithStoresDescriptorsRecorderForwarder(
		nil,
		mappings,
		files.NewMemoryStore(),
		grpcdesc.NewStore(),
		recording.NewStore(),
		proxy.NewForwarder(nil),
	)
	createMapping(t, handler, `{
	  "request": {
	    "method": "GET",
	    "urlPath": "/served"
	  },
	  "response": {
	    "status": 200,
	    "headers": {
	      "X-Served": "yes"
	    },
	    "body": "served"
	  }
	}`)

	served := requestWithHeadersAndBody(t, handler, http.MethodGet, "/served", map[string]string{"X-Capture": "one"}, "")
	if served.Code != http.StatusOK {
		t.Fatalf("served status = %d, want 200", served.Code)
	}

	snapshot := requestWithBody(t, handler, http.MethodPost, "/__admin/recordings/snapshot", `{
	  "captureHeaders": {
	    "X-Capture": {}
	  },
	  "persist": true
	}`)
	if snapshot.Code != http.StatusOK {
		t.Fatalf("snapshot status = %d, want 200: %s", snapshot.Code, snapshot.Body.String())
	}
	assertSnapshotTotal(t, snapshot.Body.Bytes(), 1)
	if mappings.Count() != 2 {
		t.Fatalf("mappings count = %d, want original + snapshot", mappings.Count())
	}
}

func assertSnapshotTotal(t *testing.T, body []byte, want int) {
	t.Helper()

	var response struct {
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode snapshot response %q: %v", body, err)
	}
	if response.Meta.Total != want {
		t.Fatalf("snapshot total = %d, want %d", response.Meta.Total, want)
	}
}

type recordingRoundTripFunc func(*http.Request) (*http.Response, error)

func (f recordingRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
