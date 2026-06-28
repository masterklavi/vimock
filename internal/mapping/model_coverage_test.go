package mapping

import (
	"encoding/json"
	"net/http"
	"testing"

	"vimock/internal/matcher"
)

func TestMappingAccessorsAndRuntimeResponse(t *testing.T) {
	stub := parseMappingJSON(t, `{
	  "scenarioName": "checkout",
	  "requiredScenarioState": "Started",
	  "newScenarioState": "Paid",
	  "priority": 2,
	  "request": {
	    "method": "POST",
	    "urlPath": "/checkout",
	    "queryParameters": {"source": {"equalTo": "mobile"}},
	    "bodyPatterns": [{"matchesJsonPath": "$.id"}]
	  },
	  "response": {
	    "status": 201,
	    "headers": {"X-Step": ["one", "two"]},
	    "body": "created {{jsonPath request.body '$.id'}}",
	    "transformers": ["response-template"]
	  }
	}`)

	if stub.Priority() != 2 || stub.RequiredScenarioState() != "Started" || stub.NewScenarioState() != "Paid" {
		t.Fatalf("unexpected accessors: priority=%d required=%q new=%q", stub.Priority(), stub.RequiredScenarioState(), stub.NewScenarioState())
	}
	request := stub.Request()
	if !request.UsesQuery() || !request.RequiresBody() {
		t.Fatalf("request flags = query:%v body:%v, want true/true", request.UsesQuery(), request.RequiresBody())
	}
	response := stub.RuntimeResponse()
	if !response.UsesResponseTemplate() || !response.RequiresRequestBody() || response.Template == nil {
		t.Fatalf("response template flags not set: %+v", response)
	}

	cloned := stub.Response()
	cloned.Headers["X-Step"][0] = "mutated"
	cloned.Body[0] = 'X'
	again := stub.Response()
	if again.Headers["X-Step"][0] != "one" || string(again.Body) != "created {{jsonPath request.body '$.id'}}" {
		t.Fatalf("Response() did not protect stored response: headers=%v body=%q", again.Headers, again.Body)
	}
}

func TestWithIDAndRequestMatching(t *testing.T) {
	stub := parseMappingJSON(t, `{
	  "request": {
	    "method": "ANY",
	    "urlPattern": "/items/[0-9]+",
	    "headers": {"X-Trace": {"equalTo": "abc"}}
	  },
	  "response": {"status": 200}
	}`)
	withID, err := stub.WithID("11111111-1111-4111-8111-111111111111")
	if err != nil {
		t.Fatalf("WithID(): %v", err)
	}
	if withID.ID() != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("id = %q", withID.ID())
	}
	if _, err := stub.WithID("bad"); err == nil {
		t.Fatal("WithID(bad) error = nil, want error")
	}

	request := withID.Request()
	if !request.Matches(http.MethodDelete, "/items/42", "/items/42", nil, http.Header{"X-Trace": []string{"abc"}}, matcher.NewBodyContext(nil)) {
		t.Fatal("request did not match ANY/urlPattern/header")
	}
	if request.Matches(http.MethodGet, "/items/no", "/items/no", nil, http.Header{"X-Trace": []string{"abc"}}, matcher.NewBodyContext(nil)) {
		t.Fatal("request matched invalid URL")
	}
}

func TestParseJSONAdditionalBranches(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "base64 body",
			body: `{
			  "request": {"method": "GET", "url": "/bin"},
			  "response": {"status": 200, "base64Body": "AAH/"}
			}`,
		},
		{
			name: "string array headers and lognormal delay",
			body: `{
			  "request": {"method": "GET", "urlPathPattern": "/slow/.*"},
			  "response": {
			    "status": 200,
			    "headers": {"X-Multi": ["a", "b"]},
			    "delayDistribution": {"type": "lognormal", "median": 50, "sigma": 0.5}
			  }
			}`,
		},
		{
			name: "uniform delay and chunked",
			body: `{
			  "request": {"method": "POST", "urlPath": "/slow"},
			  "response": {
			    "status": 202,
			    "delayDistribution": {"type": "uniform", "lower": 1, "upper": 2},
			    "chunkedDribbleDelay": {"numberOfChunks": 2, "totalDuration": 4}
			  }
			}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseJSON([]byte(tt.body)); err != nil {
				t.Fatalf("ParseJSON(): %v", err)
			}
		})
	}
}

func TestParseJSONRejectsAdditionalInvalidBranches(t *testing.T) {
	tests := []string{
		`{"request":{"method":"GET","urlPattern":"["},"response":{"status":200}}`,
		`{"request":{"method":"GET","url":"/"},"response":{"status":99}}`,
		`{"request":{"method":"GET","url":"/"},"response":{"headers":{"X":{}}}}`,
		`{"request":{"method":"GET","url":"/"},"response":{"base64Body":"not-base64"}}`,
		`{"request":{"method":"GET","url":"/"},"response":{"body":{}}}`,
		`{"request":{"method":"GET","url":"/"},"response":{"transformers":"response-template"}}`,
		`{"request":{"method":"GET","url":"/"},"response":{"fixedDelayMilliseconds":-1}}`,
		`{"request":{"method":"GET","url":"/"},"response":{"delayDistribution":{"type":"uniform","lower":2,"upper":1}}}`,
		`{"request":{"method":"GET","url":"/"},"response":{"delayDistribution":{"type":"unknown"}}}`,
		`{"request":{"method":"GET","url":"/"},"response":{"chunkedDribbleDelay":{"numberOfChunks":0,"totalDuration":1}}}`,
	}
	for _, body := range tests {
		t.Run(body, func(t *testing.T) {
			if _, err := ParseJSON([]byte(body)); err == nil {
				t.Fatal("ParseJSON() error = nil, want error")
			}
		})
	}
}

func TestDecodeObjectRejectsTrailingJSON(t *testing.T) {
	_, err := decodeObject([]byte(`{"request":{},"response":{}} {}`))
	if err == nil {
		t.Fatal("decodeObject() error = nil, want trailing JSON error")
	}
}

func TestIsValidIDRejectsMalformedUUIDs(t *testing.T) {
	invalid := []string{
		"11111111-1111-4111-8111-11111111111",
		"111111111111-4111-8111-111111111111",
		"zzzzzzzz-1111-4111-8111-111111111111",
	}
	for _, id := range invalid {
		if IsValidID(id) {
			t.Fatalf("IsValidID(%q) = true, want false", id)
		}
	}
}

func TestMustMarshalRawPanicsOnUnsupportedValue(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("mustMarshalRaw did not panic")
		}
	}()
	_ = mustMarshalRaw(func() {})
}

func TestCloneHelpersNilBranches(t *testing.T) {
	if cloneDelayDistribution(nil) != nil || cloneChunkedDribbleDelay(nil) != nil || cloneHeaders(nil) != nil || cloneResponseTemplate(nil) != nil || cloneBytes(nil) != nil {
		t.Fatal("nil clone helper returned non-nil")
	}
}

func TestResponseTemplateInvalidJSONPathFallsBackToRuntimeTemplate(t *testing.T) {
	stub := parseMappingJSON(t, `{
	  "request": {"method": "POST", "urlPath": "/template"},
	  "response": {
	    "status": 200,
	    "body": "{{jsonPath request.body 'not-jsonpath'}}",
	    "transformers": ["response-template"]
	  }
	}`)
	response := stub.RuntimeResponse()
	if response.Template != nil {
		t.Fatal("invalid compiled template should fall back to runtime regex renderer")
	}
	if !response.UsesResponseTemplate() || !response.RequiresRequestBody() {
		t.Fatal("response-template flags should still be true")
	}
}

func TestRequestPatternMatchesLazyLoadsQueryAndBodyOnlyWhenNeeded(t *testing.T) {
	stub := parseMappingJSON(t, `{
	  "request": {
	    "method": "POST",
	    "urlPath": "/items",
	    "queryParameters": {"q": {"equalTo": "1"}},
	    "bodyPatterns": [{"equalToJson": {"id":1}}]
	  },
	  "response": {"status": 200}
	}`)
	request := stub.Request()
	queryCalls := 0
	bodyCalls := 0
	matched := request.MatchesLazy(http.MethodPost, "/items?q=1", "/items", func() map[string][]string {
		queryCalls++
		return map[string][]string{"q": {"1"}}
	}, nil, func() *matcher.BodyContext {
		bodyCalls++
		return matcher.NewBodyContext([]byte(`{"id":1}`))
	})
	if !matched || queryCalls != 1 || bodyCalls != 1 {
		t.Fatalf("matched=%v queryCalls=%d bodyCalls=%d", matched, queryCalls, bodyCalls)
	}
}

func TestMarshalJSONIncludesUpdatedID(t *testing.T) {
	stub := parseMappingJSON(t, `{"request":{"method":"GET","url":"/"},"response":{"status":200}}`)
	withID, err := stub.WithID("11111111-1111-4111-8111-111111111111")
	if err != nil {
		t.Fatalf("WithID(): %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(mustJSON(t, withID), &raw); err != nil {
		t.Fatalf("Unmarshal(): %v", err)
	}
	if raw["id"] != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("id = %v", raw["id"])
	}
}

func TestParseJSONRejectsInvalidTypedFields(t *testing.T) {
	tests := []string{
		`{"persistent":"true","request":{"method":"GET","url":"/"},"response":{"status":200}}`,
		`{"priority":"1","request":{"method":"GET","url":"/"},"response":{"status":200}}`,
		`{"scenarioName":1,"request":{"method":"GET","url":"/"},"response":{"status":200}}`,
		`{"request":{"method":1,"url":"/"},"response":{"status":200}}`,
		`{"request":{"method":"GET","urlPathPattern":"["},"response":{"status":200}}`,
		`{"request":{"method":"GET","url":"/","bodyPatterns":{}},"response":{"status":200}}`,
		`{"request":{"method":"GET","url":"/","queryParameters":{"q":{"equalTo":"1"}}},"response":{"status":200,"delayDistribution":{"type":"lognormal","median":0,"sigma":0}}}`,
	}
	for _, body := range tests {
		t.Run(body, func(t *testing.T) {
			if _, err := ParseJSON([]byte(body)); err == nil {
				t.Fatal("ParseJSON() error = nil")
			}
		})
	}
}

func TestRequestPatternNegativeMatches(t *testing.T) {
	stub := parseMappingJSON(t, `{"request":{"method":"GET","urlPath":"/items"},"response":{"status":200}}`)
	request := stub.Request()
	if request.Matches("", "/items", "/items", nil, nil, nil) {
		t.Fatal("empty method should not match GET mapping")
	}
	if request.Matches("GET", "/other", "/other", nil, nil, nil) {
		t.Fatal("different path should not match")
	}
	stub = parseMappingJSON(t, `{"request":{"method":"ANY","url":"/items?q=1"},"response":{"status":200}}`)
	if !stub.Request().Matches("PATCH", "/items?q=1", "/items", nil, nil, nil) {
		t.Fatal("ANY/url exact should match")
	}
}

func TestParseJSONCoversResponseBodyAndHeaderVariants(t *testing.T) {
	stub := parseMappingJSON(t, `{
	  "name": "json-body",
	  "persistent": true,
	  "metadata": {"team": "qa"},
	  "request": {"method": "GET", "urlPathPattern": "/items/[0-9]+"},
	  "response": {
	    "status": 204,
	    "headers": {"X-One": "value"},
	    "jsonBody": {"ok": true}
	  }
	}`)
	if !stub.Persistent() || stub.Name() != "json-body" {
		t.Fatalf("metadata flags not parsed: persistent=%v name=%q", stub.Persistent(), stub.Name())
	}
	response := stub.Response()
	if !response.JSON || string(response.Body) != `{"ok": true}` || response.Headers["X-One"][0] != "value" {
		t.Fatalf("response variants not parsed: %+v body=%s", response, response.Body)
	}
	if !stub.Request().Matches(http.MethodGet, "/items/42?ignored=1", "/items/42", nil, nil, nil) {
		t.Fatal("urlPathPattern should match full request path")
	}
	if stub.Request().Matches(http.MethodGet, "/items/no", "/items/no", nil, nil, nil) {
		t.Fatal("urlPathPattern should reject non-matching path")
	}

	stub = parseMappingJSON(t, `{"request":{"method":"GET","url":"/null"},"response":{"body":null}}`)
	if stub.Response().Body != nil {
		t.Fatal("null response body should stay nil")
	}
}

func TestParseJSONRejectsMoreInvalidScalarBranches(t *testing.T) {
	tests := []string{
		`{"id":1,"request":{"method":"GET","url":"/"},"response":{"status":200}}`,
		`{"request":{"method":"GET","url":1},"response":{"status":200}}`,
		`{"request":{"method":"GET","url":"/","headers":{"X":"bad"}},"response":{"status":200}}`,
		`{"request":{"method":"GET","url":"/"},"response":{"status":"200"}}`,
		`{"request":{"method":"GET","url":"/"},"response":{"proxyBaseUrl":1}}`,
		`{"request":{"method":"GET","url":"/"},"response":{"proxyUrlPrefixToRemove":1}}`,
		`{"request":{"method":"GET","url":"/"},"response":{"delayDistribution":{"type":"lognormal","median":1,"sigma":"bad"}}}`,
		`{"request":{"method":"GET","url":"/"},"response":{"chunkedDribbleDelay":{"numberOfChunks":"bad","totalDuration":1}}}`,
		`{"request":{"method":"GET","url":"/"},"response":{"chunkedDribbleDelay":{"numberOfChunks":1,"totalDuration":"bad"}}}`,
		`{"request":{"method":"GET","url":"/"},"response":{"chunkedDribbleDelay":[]}}`,
	}
	for _, body := range tests {
		t.Run(body, func(t *testing.T) {
			if _, err := ParseJSON([]byte(body)); err == nil {
				t.Fatal("ParseJSON() error = nil")
			}
		})
	}
}

func TestRequestPatternMatchesLazyShortCircuits(t *testing.T) {
	stub := parseMappingJSON(t, `{
	  "request": {
	    "method": "POST",
	    "urlPath": "/items",
	    "queryParameters": {"q": {"equalTo": "1"}},
	    "bodyPatterns": [{"matchesJsonPath": "$.id"}]
	  },
	  "response": {"status": 200}
	}`)
	request := stub.Request()

	queryCalls := 0
	bodyCalls := 0
	if request.MatchesLazy(http.MethodGet, "/items?q=1", "/items", func() map[string][]string {
		queryCalls++
		return map[string][]string{"q": {"1"}}
	}, nil, func() *matcher.BodyContext {
		bodyCalls++
		return matcher.NewBodyContext([]byte(`{"id":1}`))
	}) {
		t.Fatal("method mismatch should not match")
	}
	if queryCalls != 0 || bodyCalls != 0 {
		t.Fatalf("lazy providers called too early: query=%d body=%d", queryCalls, bodyCalls)
	}

	if request.MatchesLazy(http.MethodPost, "/items?q=2", "/items", func() map[string][]string {
		queryCalls++
		return map[string][]string{"q": {"2"}}
	}, nil, func() *matcher.BodyContext {
		bodyCalls++
		return matcher.NewBodyContext([]byte(`{"id":1}`))
	}) {
		t.Fatal("query mismatch should not match")
	}
	if queryCalls != 1 || bodyCalls != 0 {
		t.Fatalf("query mismatch should not load body: query=%d body=%d", queryCalls, bodyCalls)
	}

	if request.MatchesLazy(http.MethodPost, "/items?q=1", "/items", func() map[string][]string {
		return map[string][]string{"q": {"1"}}
	}, nil, func() *matcher.BodyContext {
		bodyCalls++
		return nil
	}) {
		t.Fatal("nil body should not match body-required mapping")
	}
	if bodyCalls != 1 {
		t.Fatalf("body provider calls = %d, want 1", bodyCalls)
	}
}
