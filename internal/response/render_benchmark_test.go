package response

import (
	"bytes"
	"net/http"
	"testing"

	"vimock/internal/mapping"
	"vimock/internal/matcher"
)

func BenchmarkRenderTemplateJSONPath(b *testing.B) {
	renderer := NewRenderer(nil)
	definition := mapping.ResponseDefinition{
		Status:       http.StatusOK,
		Body:         []byte(`{"id":"{{jsonPath request.body '$.id'}}","nested":"{{jsonPath request.body '$.payload.value'}}"}`),
		JSON:         true,
		Transformers: []string{"response-template"},
	}
	requestBody := []byte(`{"id":"rpc-42","payload":{"value":"vimock"}}`)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rendered, err := renderer.Render(definition, matcher.NewBodyContext(requestBody))
		if err != nil {
			b.Fatalf("Render(): %v", err)
		}
		if !bytes.Contains(rendered.Body, []byte(`rpc-42`)) {
			b.Fatalf("rendered body = %s", rendered.Body)
		}
	}
}
