package response

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"vimock/internal/files"
	"vimock/internal/mapping"
	"vimock/internal/matcher"
)

func TestTemplateAndBodyFiles(t *testing.T) {
	t.Run("json body templating", func(t *testing.T) {
		renderer := NewRenderer(nil)
		rendered, err := renderer.Render(mapping.ResponseDefinition{
			Status:       http.StatusOK,
			Body:         []byte(`{"requestId":"{{jsonPath request.body '$.requestId'}}"}`),
			JSON:         true,
			Transformers: []string{"response-template"},
		}, matcher.NewBodyContext([]byte(`{"requestId":"req-\"123"}`)))
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}

		var body map[string]string
		if err := json.Unmarshal(rendered.Body, &body); err != nil {
			t.Fatalf("decode rendered body: %v", err)
		}
		if body["requestId"] != `req-"123` {
			t.Fatalf("requestId = %q, want req-\"123", body["requestId"])
		}
	})

	t.Run("string body templating", func(t *testing.T) {
		renderer := NewRenderer(nil)
		rendered, err := renderer.Render(mapping.ResponseDefinition{
			Status:       http.StatusOK,
			Body:         []byte(`hello {{jsonPath request.body '$.name'}}`),
			Transformers: []string{"response-template"},
		}, matcher.NewBodyContext([]byte(`{"name":"vimock"}`)))
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		if got := string(rendered.Body); got != "hello vimock" {
			t.Fatalf("body = %q, want hello vimock", got)
		}
	})

	t.Run("json rpc id echo", func(t *testing.T) {
		renderer := NewRenderer(nil)
		rendered, err := renderer.Render(mapping.ResponseDefinition{
			Status:       http.StatusOK,
			Body:         []byte(`{"jsonrpc":"2.0","id":"{{jsonPath request.body '$.id'}}"}`),
			JSON:         true,
			Transformers: []string{"response-template"},
		}, matcher.NewBodyContext([]byte(`{"jsonrpc":"2.0","id":"rpc-42"}`)))
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}

		var body map[string]string
		if err := json.Unmarshal(rendered.Body, &body); err != nil {
			t.Fatalf("decode rendered body: %v", err)
		}
		if body["id"] != "rpc-42" {
			t.Fatalf("id = %q, want rpc-42", body["id"])
		}
	})

	t.Run("body file bytes unchanged", func(t *testing.T) {
		fileStore := files.NewMemoryStore()
		want := []byte{0x25, 0x50, 0x44, 0x46, 0x00, 0xff, 0x10, 0x7f}
		fileStore.Put("document.pdf", want)

		renderer := NewRenderer(fileStore)
		rendered, err := renderer.Render(mapping.ResponseDefinition{
			Status:       http.StatusOK,
			BodyFile:     "document.pdf",
			Transformers: []string{"response-template"},
		}, matcher.NewBodyContext([]byte(`{"id":"ignored"}`)))
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		if !bytes.Equal(rendered.Body, want) {
			t.Fatalf("body bytes = %v, want %v", rendered.Body, want)
		}
	})
}

func TestBodyFileMissingReturnsError(t *testing.T) {
	renderer := NewRenderer(files.NewMemoryStore())
	_, err := renderer.Render(mapping.ResponseDefinition{
		Status:   http.StatusOK,
		BodyFile: "missing.pdf",
	}, matcher.NewBodyContext(nil))
	if err == nil {
		t.Fatalf("Render() error = nil, want missing body file error")
	}
}
