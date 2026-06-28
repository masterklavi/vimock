package matcher

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
)

func TestJSONPathCurrentMockPatterns(t *testing.T) {
	testCurrentMockPatterns(t)
}

func TestCurrentMockPatterns(t *testing.T) {
	testCurrentMockPatterns(t)
}

func testCurrentMockPatterns(t *testing.T) {
	body := mustParseJSON(t, []byte(`{
	  "method": "rests.get",
	  "guids": ["b27ed95d-3717-4538-9be6-a7136b8ad52f"],
	  "params": {
	    "provider_strict": true,
	    "providers": ["provider-1"],
	    "destinations": ["destination-1", "destination-2"],
	    "chains": [
	      ["source", "middle", "destination"],
	      {"chain_nodes": ["a", "b", "c"]}
	    ],
	    "pickup_dates": ["2026-01-01T00:00:00Z"],
	    "filter": {
	      "tripGuid": "trip-1",
	      "tripId": 42
	    },
	    "seals": [
	      [{"officeGuid": "office-1", "numbers": ["11111111", "222222222", "3333333333"]}]
	    ]
	  }
	}`))

	tests := []string{
		"$[?(@.method == 'rests.get')]",
		"$.guids[?(@ == 'b27ed95d-3717-4538-9be6-a7136b8ad52f')]",
		"$[?(@.params.provider_strict == true)]",
		"$.params.providers[?(@ == 'provider-1')]",
		"$.params[?(@.destinations.size() == 2)]",
		"$.params.chains[?(@ == ['source','middle','destination'])]",
		"$.params.chains.*[?(@.chain_nodes == ['a','b','c'])]",
		"$.params.pickup_dates[?(@ == '2026-01-01T00:00:00Z')]",
		"$.params.filter[?(@.tripGuid == 'trip-1')]",
		"$.params.filter[?(@.tripId == 42)]",
		"$.params.seals[0][?(@.officeGuid == 'office-1')]",
		"$.params.seals[0][?(@.numbers == ['11111111', '222222222', '3333333333'])]",
	}

	for _, expression := range tests {
		t.Run(expression, func(t *testing.T) {
			compiled, err := CompileJSONPath(expression)
			if err != nil {
				t.Fatalf("CompileJSONPath() error = %v", err)
			}
			if !compiled.Exists(body) {
				t.Fatalf("%s did not match", expression)
			}
		})
	}
}

func TestJSONPathAbsent(t *testing.T) {
	body := mustParseJSON(t, []byte(`{"params": {"chains": []}}`))

	compiled, err := CompileJSONPath("$.params.providers")
	if err != nil {
		t.Fatalf("CompileJSONPath() error = %v", err)
	}
	if compiled.Exists(body) {
		t.Fatalf("providers exists = true, want false")
	}
}

func TestBodyPatternMatchesJsonPathAndEqualToJSON(t *testing.T) {
	pattern, err := ParseBodyPattern(json.RawMessage(`{"matchesJsonPath": "$.items[?(@.id == '1')]"}`))
	if err != nil {
		t.Fatalf("ParseBodyPattern(matchesJsonPath) error = %v", err)
	}
	if !MatchBodyPatterns([]BodyPattern{pattern}, []byte(`{"items":[{"id":"1"}]}`)) {
		t.Fatalf("matchesJsonPath pattern did not match")
	}

	absent, err := ParseBodyPattern(json.RawMessage(`{"matchesJsonPath": {"expression": "$.missing", "absent": true}}`))
	if err != nil {
		t.Fatalf("ParseBodyPattern(absent) error = %v", err)
	}
	if !MatchBodyPatterns([]BodyPattern{absent}, []byte(`{"items":[{"id":"1"}]}`)) {
		t.Fatalf("absent pattern did not match")
	}

	equalToJSON, err := ParseBodyPattern(json.RawMessage(`{"equalToJson": "{\"b\":2,\"a\":1}"}`))
	if err != nil {
		t.Fatalf("ParseBodyPattern(equalToJson) error = %v", err)
	}
	if !MatchBodyPatterns([]BodyPattern{equalToJSON}, []byte(`{"a":1,"b":2}`)) {
		t.Fatalf("equalToJson pattern did not match")
	}
}

func TestEqualToMatchers(t *testing.T) {
	queryMatchers, err := ParseEqualToMap(json.RawMessage(`{"date":{"equalTo":"2025-10-14"}}`), "request.queryParameters")
	if err != nil {
		t.Fatalf("ParseEqualToMap(query) error = %v", err)
	}
	query := url.Values{"date": []string{"2025-10-14"}}
	if !MatchQuery(queryMatchers, query) {
		t.Fatalf("query matcher did not match")
	}

	headerMatchers, err := ParseEqualToMap(json.RawMessage(`{"Content-Type":{"equalTo":"application/protobuf"}}`), "request.headers")
	if err != nil {
		t.Fatalf("ParseEqualToMap(headers) error = %v", err)
	}
	headers := http.Header{"Content-Type": []string{"application/protobuf"}}
	if !MatchHeaders(headerMatchers, headers) {
		t.Fatalf("header matcher did not match")
	}
}

func mustParseJSON(t *testing.T, body []byte) any {
	t.Helper()

	value, err := ParseJSON(body)
	if err != nil {
		t.Fatalf("ParseJSON() error = %v", err)
	}
	return value
}
