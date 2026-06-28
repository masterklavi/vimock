package matcher

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
)

func TestRequestMatcherNegativeBranches(t *testing.T) {
	invalidPatterns := []json.RawMessage{
		json.RawMessage(`[]`),
		json.RawMessage(`{"unknown": true}`),
		json.RawMessage(`{"matchesJsonPath": {}}`),
		json.RawMessage(`{"equalToJson": "not-json"}`),
	}
	for _, raw := range invalidPatterns {
		if _, err := ParseBodyPattern(raw); err == nil {
			t.Fatalf("ParseBodyPattern(%s) error = nil", raw)
		}
	}
	if _, err := ParseEqualToMap(json.RawMessage(`[]`), "query"); err == nil {
		t.Fatal("ParseEqualToMap array error = nil")
	}
	if _, err := ParseEqualToMap(json.RawMessage(`{"x": []}`), "query"); err == nil {
		t.Fatal("ParseEqualToMap matcher array error = nil")
	}
	if _, err := ParseEqualToMap(json.RawMessage(`{"x": {}}`), "query"); err == nil {
		t.Fatal("ParseEqualToMap missing equalTo error = nil")
	}
	if _, err := ParseEqualToMap(json.RawMessage(`{"x": {"equalTo": 1}}`), "query"); err == nil {
		t.Fatal("ParseEqualToMap non-string equalTo error = nil")
	}

	patterns := map[string]EqualTo{"q": {Expected: "2"}}
	if MatchQuery(patterns, url.Values{"q": {"1"}}) || MatchQuery(patterns, nil) {
		t.Fatal("query matcher should not match")
	}
	headers := map[string]EqualTo{"X-Trace": {Expected: "2"}}
	if MatchHeaders(headers, http.Header{"X-Trace": {"1"}}) || MatchHeaders(headers, nil) {
		t.Fatal("header matcher should not match")
	}
	if MatchBodyPatternsWithContext([]BodyPattern{{equalToJSON: []byte(`{"id":1}`)}}, NewBodyContext([]byte(`not-json`))) {
		t.Fatal("invalid actual JSON should not match equalToJson")
	}
	if NewBodyContext(nil).Raw() != nil || (*BodyContext)(nil).Raw() != nil {
		t.Fatal("Raw nil branch mismatch")
	}
}
