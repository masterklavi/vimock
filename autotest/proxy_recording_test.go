package autotest

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestBlackBoxProxyAndRecordingWorkflow(t *testing.T) {
	s := requireTarget(t)
	suffix := uniqueName(t)
	base := "/autotest/" + suffix
	var upstreamCalls int64

	_, upstreamURL := newReachableUpstream(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&upstreamCalls, 1)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Upstream", "ok")
		switch {
		case r.URL.Path == "/proxied":
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"source":"proxy","query":"` + r.URL.RawQuery + `"}`))
		case strings.HasSuffix(r.URL.Path, "/record-source"):
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"source":"recording","path":"` + r.URL.Path + `"}`))
		default:
			http.NotFound(w, r)
		}
	}))

	createMapping(t, s, map[string]any{
		"name":       "autotest-proxy-" + suffix,
		"persistent": true,
		"priority":   10,
		"request": map[string]any{
			"method":     "ANY",
			"urlPattern": base + "/proxy/.*",
		},
		"response": map[string]any{
			"status":                    200,
			"proxyBaseUrl":              upstreamURL,
			"proxyUrlPrefixToRemove":    base + "/proxy",
			"fixedDelayMilliseconds":    1,
			"delayDistribution":         map[string]any{"type": "uniform", "lower": 1, "upper": 1},
			"chunkedDribbleDelay":       map[string]any{"numberOfChunks": 2, "totalDuration": 1},
			"ignoredWireMockCompatible": "preserved",
		},
	})

	proxyResp, proxyBody := s.request(t, http.MethodGet, base+"/proxy/proxied?sku=1", nil, nil)
	expectStatus(t, proxyResp, proxyBody, http.StatusAccepted)
	if !strings.Contains(string(proxyBody), `"source":"proxy"`) || !strings.Contains(string(proxyBody), "sku=1") {
		t.Fatalf("proxy body = %s", proxyBody)
	}
	if got := proxyResp.Header.Get("X-Upstream"); got != "ok" {
		t.Fatalf("proxied X-Upstream = %q, want ok", got)
	}

	startResp, startBody := s.requestJSON(t, http.MethodPost, "/__admin/recordings/start", map[string]any{
		"targetBaseUrl":      upstreamURL,
		"captureHeaders":     map[string]any{"X-Request-Id": map[string]any{}},
		"requestBodyPattern": "equalToJson",
		"persist":            true,
	})
	expectStatus(t, startResp, startBody, http.StatusOK)

	recordedPath := base + "/record-source"
	recordedResp, recordedBody := s.request(t, http.MethodGet, recordedPath, nil, map[string]string{"X-Request-Id": "req-1"})
	expectStatus(t, recordedResp, recordedBody, http.StatusCreated)
	if !strings.Contains(string(recordedBody), `"source":"recording"`) {
		t.Fatalf("recorded proxy body = %s", recordedBody)
	}

	stopResp, stopBody := s.request(t, http.MethodPost, "/__admin/recordings/stop", nil, nil)
	expectStatus(t, stopResp, stopBody, http.StatusOK)
	assertSnapshotHasMappings(t, stopBody, 1)

	callsAfterStop := atomic.LoadInt64(&upstreamCalls)
	playbackResp, playbackBody := s.request(t, http.MethodGet, recordedPath, nil, map[string]string{"X-Request-Id": "req-1"})
	expectStatus(t, playbackResp, playbackBody, http.StatusCreated)
	if !strings.Contains(string(playbackBody), `"source":"recording"`) {
		t.Fatalf("recording playback body = %s", playbackBody)
	}
	if atomic.LoadInt64(&upstreamCalls) != callsAfterStop {
		t.Fatalf("recording playback called upstream again")
	}

	snapshotResp, snapshotBody := s.requestJSON(t, http.MethodPost, "/__admin/recordings/snapshot", map[string]any{
		"captureHeaders":     map[string]any{"X-Request-Id": map[string]any{}},
		"requestBodyPattern": "equalToJson",
		"persist":            true,
	})
	expectStatus(t, snapshotResp, snapshotBody, http.StatusOK)
	assertSnapshotHasMappings(t, snapshotBody, 1)
}

func assertSnapshotHasMappings(t *testing.T, body []byte, minTotal int) {
	t.Helper()
	var response struct {
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode snapshot %q: %v", body, err)
	}
	if response.Meta.Total < minTotal {
		t.Fatalf("snapshot total = %d, want >= %d", response.Meta.Total, minTotal)
	}
}
