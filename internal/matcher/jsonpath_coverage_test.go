package matcher

import "testing"

func TestJSONPathValuesAndFirstValueBranches(t *testing.T) {
	body := mustParseJSON(t, []byte(`{"items":[{"id":1},{"id":2}],"object":{"b":2,"a":1},"text":"abc"}`))
	compiled, err := CompileJSONPath("$.items.*")
	if err != nil {
		t.Fatalf("CompileJSONPath(): %v", err)
	}
	if values := compiled.Values(body); len(values) != 2 {
		t.Fatalf("Values len = %d, want 2", len(values))
	}
	if value, ok := compiled.FirstValue(body); !ok || value == nil {
		t.Fatalf("FirstValue wildcard = %v %v", value, ok)
	}
	compiled, _ = CompileJSONPath("$.object.*")
	if value, ok := compiled.FirstValue(body); !ok || value == nil {
		t.Fatalf("FirstValue map wildcard = %v %v", value, ok)
	}
	compiled, _ = CompileJSONPath("$.missing.*")
	if _, ok := compiled.FirstValue(body); ok {
		t.Fatal("FirstValue missing ok = true")
	}
	compiled, _ = CompileJSONPath("$.items[9]")
	if _, ok := compiled.FirstValue(body); ok {
		t.Fatal("FirstValue bad index ok = true")
	}
	compiled, _ = CompileJSONPath("$.text[0]")
	if _, ok := compiled.FirstValue(body); ok {
		t.Fatal("FirstValue non-array index ok = true")
	}
	compiled, _ = CompileJSONPath("$.items[?(@.id == 2)]")
	if value, ok := compiled.FirstValue(body); !ok || value == nil {
		t.Fatalf("FirstValue filter = %v %v", value, ok)
	}
}

func TestJSONPathValueSizeAndNormalizeBranches(t *testing.T) {
	body := mustParseJSON(t, []byte(`{"items":[1,2],"object":{"a":1},"text":"abc","float":1.0}`))
	expressions := []string{
		"$[?(@.items.size() == 2)]",
		"$[?(@.object.size() == 1)]",
		"$[?(@.text.size() == 3)]",
	}
	for _, expression := range expressions {
		compiled, err := CompileJSONPath(expression)
		if err != nil {
			t.Fatalf("CompileJSONPath(%s): %v", expression, err)
		}
		if !compiled.Exists(body) {
			t.Fatalf("%s did not match", expression)
		}
	}
	compiled, _ := CompileJSONPath("$[?(@.float.size() == 1)]")
	if compiled.Exists(body) {
		t.Fatal("size() on number should not match")
	}
}

func TestJSONPathCompileAndParseErrors(t *testing.T) {
	expressions := []string{"", "items", "$[?(@.id == 1)", "$.", "$.items[bad]", "$#", "$[?(.id == 1)]", "$[?(@.id != 1)]", "$[?(@.id == nope']"}
	for _, expression := range expressions {
		if _, err := CompileJSONPath(expression); err == nil {
			t.Fatalf("CompileJSONPath(%q) error = nil", expression)
		}
	}
}
