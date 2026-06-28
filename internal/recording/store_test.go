package recording

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestBuildSnapshotUsesBase64BodyForBinaryResponses(t *testing.T) {
	snapshot, err := BuildSnapshot([]ServeEvent{
		{
			Method:         http.MethodGet,
			Path:           "/binary",
			ResponseStatus: http.StatusOK,
			ResponseHeaders: http.Header{
				"Content-Type": []string{"application/octet-stream"},
			},
			ResponseBody: []byte{0x00, 0x01, 0xff},
		},
	}, Spec{})
	if err != nil {
		t.Fatalf("BuildSnapshot() error = %v", err)
	}
	if snapshot.Meta.Total != 1 || len(snapshot.Mappings) != 1 {
		t.Fatalf("snapshot = %+v, want one mapping", snapshot)
	}

	body, err := json.Marshal(snapshot.Mappings[0])
	if err != nil {
		t.Fatalf("marshal mapping: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("decode mapping: %v", err)
	}
	response := raw["response"].(map[string]any)
	if response["base64Body"] != "AAH/" {
		t.Fatalf("base64Body = %v, want AAH/", response["base64Body"])
	}
}

func TestStoreStartStopValidation(t *testing.T) {
	store := NewStore()
	if err := store.Start(Spec{}); err == nil {
		t.Fatalf("Start() error = nil, want targetBaseUrl required")
	}
	if _, err := store.Stop(); err == nil {
		t.Fatalf("Stop() error = nil, want not running")
	}
	if err := store.Start(Spec{TargetBaseURL: "http://upstream.local"}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	store.AddRecordingEvent(ServeEvent{
		Method:         http.MethodGet,
		Path:           "/items",
		ResponseStatus: http.StatusNoContent,
	})
	snapshot, err := store.Stop()
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if snapshot.Meta.Total != 1 {
		t.Fatalf("snapshot total = %d, want 1", snapshot.Meta.Total)
	}
}
