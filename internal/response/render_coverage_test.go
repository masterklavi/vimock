package response

import (
	"net/http"
	"testing"

	"vimock/internal/mapping"
	"vimock/internal/matcher"
)

type stringerValue struct{}

func (stringerValue) String() string { return "stringer" }

func TestRenderUsesCompiledTemplateAndNilRequestBody(t *testing.T) {
	stub, err := mapping.ParseJSON([]byte(`{
	  "request": {"method": "POST", "url": "/template"},
	  "response": {
	    "status": 200,
	    "jsonBody": {"id":"{{jsonPath request.body '$.id'}}"},
	    "transformers": ["response-template"]
	  }
	}`))
	if err != nil {
		t.Fatalf("ParseJSON(): %v", err)
	}
	rendered, err := NewRenderer(nil).Render(stub.RuntimeResponse(), matcher.NewBodyContext([]byte(`{"id":"abc"}`)))
	if err != nil {
		t.Fatalf("Render(): %v", err)
	}
	if string(rendered.Body) != `{"id":"abc"}` {
		t.Fatalf("body = %s", rendered.Body)
	}

	rendered, err = NewRenderer(nil).Render(stub.RuntimeResponse(), nil)
	if err != nil {
		t.Fatalf("Render(nil body): %v", err)
	}
	if string(rendered.Body) != `` {
		t.Fatalf("body with nil request = %q, want empty rendered value", rendered.Body)
	}
}

func TestRenderFallbackTemplateInvalidExpressionAndMissingPath(t *testing.T) {
	renderer := NewRenderer(nil)
	rendered, err := renderer.Render(mapping.ResponseDefinition{
		Status:       http.StatusOK,
		Body:         []byte(`before {{jsonPath request.body 'not-jsonpath'}} after {{jsonPath request.body '$.missing'}}`),
		Transformers: []string{"response-template"},
	}, matcher.NewBodyContext([]byte(`{"id":"abc"}`)))
	if err != nil {
		t.Fatalf("Render(): %v", err)
	}
	if string(rendered.Body) != "before  after " {
		t.Fatalf("body = %q", rendered.Body)
	}
}

func TestStringifyVariants(t *testing.T) {
	if stringify(nil) != "" {
		t.Fatal("nil stringify should be empty")
	}
	if stringify("text") != "text" {
		t.Fatal("string stringify mismatch")
	}
	if stringify(stringerValue{}) != "stringer" {
		t.Fatal("stringer stringify mismatch")
	}
	if stringify(map[string]any{"a": 1}) != `{"a":1}` {
		t.Fatalf("map stringify mismatch: %q", stringify(map[string]any{"a": 1}))
	}
	if got := stringify(make(chan int)); got == "" {
		t.Fatal("fallback stringify should return non-empty value")
	}
}
