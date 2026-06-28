package mapping

import (
	"encoding/json"
	"strings"
	"testing"
)

const validMappingJSON = `{
  "name": "Basic Resource",
  "persistent": true,
  "request": {
    "method": "GET",
    "url": "/resource"
  },
  "response": {
    "status": 200
  },
  "priority": 1,
  "metadata": {
    "wiremock-gui": {
      "folder": "/Folder"
    }
  },
  "x-extra": {
    "enabled": true
  }
}`

func TestParseJSONGeneratesIDAndPreservesRawFields(t *testing.T) {
	stub, err := ParseJSON([]byte(validMappingJSON))
	if err != nil {
		t.Fatalf("ParseJSON() error = %v", err)
	}
	if !IsValidID(stub.ID()) {
		t.Fatalf("generated id = %q, want UUID", stub.ID())
	}
	if stub.Name() != "Basic Resource" {
		t.Fatalf("name = %q, want Basic Resource", stub.Name())
	}
	if !stub.Persistent() {
		t.Fatalf("persistent = false, want true")
	}

	var body map[string]any
	if err := json.Unmarshal(mustJSON(t, stub), &body); err != nil {
		t.Fatalf("decode marshaled mapping: %v", err)
	}
	if body["id"] != stub.ID() {
		t.Fatalf("id in JSON = %v, want %s", body["id"], stub.ID())
	}
	if _, ok := body["x-extra"]; !ok {
		t.Fatalf("unknown field x-extra was not preserved")
	}

	metadata := body["metadata"].(map[string]any)
	gui := metadata["wiremock-gui"].(map[string]any)
	if gui["folder"] != "/Folder" {
		t.Fatalf("metadata folder = %v, want /Folder", gui["folder"])
	}
}

func TestParseJSONWithIDOverridesBodyID(t *testing.T) {
	const pathID = "11111111-1111-4111-8111-111111111111"
	const bodyID = "22222222-2222-4222-8222-222222222222"

	stub, err := ParseJSONWithID([]byte(`{
	  "id": "`+bodyID+`",
	  "request": {"method": "GET", "url": "/resource"},
	  "response": {"status": 200}
	}`), pathID)
	if err != nil {
		t.Fatalf("ParseJSONWithID() error = %v", err)
	}
	if stub.ID() != pathID {
		t.Fatalf("id = %q, want path id %q", stub.ID(), pathID)
	}

	var body map[string]any
	if err := json.Unmarshal(mustJSON(t, stub), &body); err != nil {
		t.Fatalf("decode marshaled mapping: %v", err)
	}
	if body["id"] != pathID {
		t.Fatalf("id in JSON = %v, want %s", body["id"], pathID)
	}
}

func TestParseJSONResponseProxyAndDelays(t *testing.T) {
	stub, err := ParseJSON([]byte(`{
	  "request": {
	    "method": "ANY",
	    "urlPattern": "/proxy/.*"
	  },
	  "response": {
	    "status": 200,
	    "proxyBaseUrl": "https://example.com/base",
	    "proxyUrlPrefixToRemove": "/proxy",
	    "fixedDelayMilliseconds": 25,
	    "delayDistribution": {
	      "type": "uniform",
	      "lower": 10,
	      "upper": 20
	    },
	    "chunkedDribbleDelay": {
	      "numberOfChunks": 3,
	      "totalDuration": 30
	    }
	  }
	}`))
	if err != nil {
		t.Fatalf("ParseJSON() error = %v", err)
	}

	response := stub.Response()
	if response.ProxyBaseURL != "https://example.com/base" {
		t.Fatalf("ProxyBaseURL = %q, want https://example.com/base", response.ProxyBaseURL)
	}
	if response.ProxyURLPrefixToRemove != "/proxy" {
		t.Fatalf("ProxyURLPrefixToRemove = %q, want /proxy", response.ProxyURLPrefixToRemove)
	}
	if response.FixedDelayMilliseconds != 25 {
		t.Fatalf("FixedDelayMilliseconds = %d, want 25", response.FixedDelayMilliseconds)
	}
	if response.DelayDistribution == nil || response.DelayDistribution.Type != "uniform" || response.DelayDistribution.Lower != 10 || response.DelayDistribution.Upper != 20 {
		t.Fatalf("DelayDistribution = %+v, want uniform 10..20", response.DelayDistribution)
	}
	if response.ChunkedDribbleDelay == nil || response.ChunkedDribbleDelay.NumberOfChunks != 3 || response.ChunkedDribbleDelay.TotalDurationMilliseconds != 30 {
		t.Fatalf("ChunkedDribbleDelay = %+v, want 3 chunks over 30ms", response.ChunkedDribbleDelay)
	}
}

func TestParseJSONRejectsInvalidMapping(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr string
	}{
		{
			name:    "not object",
			body:    `[]`,
			wantErr: "JSON object",
		},
		{
			name:    "missing request",
			body:    `{"response":{"status":200}}`,
			wantErr: "request is required",
		},
		{
			name:    "missing response",
			body:    `{"request":{"method":"GET","url":"/resource"}}`,
			wantErr: "response is required",
		},
		{
			name:    "invalid metadata",
			body:    `{"request":{},"response":{},"metadata":[]}`,
			wantErr: "metadata must be a JSON object",
		},
		{
			name:    "invalid id",
			body:    `{"id":"not-a-uuid","request":{},"response":{}}`,
			wantErr: "not-a-uuid is not a valid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseJSON([]byte(tt.body))
			if err == nil {
				t.Fatalf("ParseJSON() error = nil, want %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return data
}
