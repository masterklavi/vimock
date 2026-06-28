package response

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"vimock/internal/files"
	"vimock/internal/mapping"
	"vimock/internal/matcher"
)

var jsonPathHelperPattern = regexp.MustCompile(`\{\{\s*jsonPath\s+request\.body\s+(?:'([^']*)'|"([^"]*)")\s*\}\}`)

type Renderer struct {
	files files.Store
}

type Rendered struct {
	Status  int
	Headers http.Header
	Body    []byte
	JSON    bool
}

func NewRenderer(fileStore files.Store) Renderer {
	if fileStore == nil {
		fileStore = files.NewMemoryStore()
	}
	return Renderer{files: fileStore}
}

func (r Renderer) Render(definition mapping.ResponseDefinition, requestBody *matcher.BodyContext) (Rendered, error) {
	rendered := Rendered{
		Status:  definition.Status,
		Headers: cloneHeaders(definition.Headers),
		JSON:    definition.JSON,
	}

	templatable := false
	switch {
	case definition.BodyFile != "":
		body, ok := r.files.Get(definition.BodyFile)
		if !ok {
			return Rendered{}, fmt.Errorf("response body file %q not found", definition.BodyFile)
		}
		rendered.Body = body
	case definition.Body != nil:
		rendered.Body = append([]byte(nil), definition.Body...)
		templatable = true
	}

	if templatable && hasResponseTemplate(definition.Transformers) && len(rendered.Body) > 0 {
		rendered.Body = renderTemplate(rendered.Body, requestBody, definition.JSON)
	}

	return rendered, nil
}

func hasResponseTemplate(transformers []string) bool {
	for _, transformer := range transformers {
		if transformer == "response-template" {
			return true
		}
	}
	return false
}

func renderTemplate(template []byte, requestBody *matcher.BodyContext, escapeJSONString bool) []byte {
	return jsonPathHelperPattern.ReplaceAllFunc(template, func(match []byte) []byte {
		parts := jsonPathHelperPattern.FindSubmatch(match)
		if len(parts) != 3 {
			return match
		}

		expression := string(parts[1])
		if expression == "" {
			expression = string(parts[2])
		}
		value, ok := jsonPathValue(requestBody, expression)
		if !ok {
			return nil
		}
		if escapeJSONString {
			value = escapeJSONStringContent(value)
		}
		return []byte(value)
	})
}

func escapeJSONStringContent(value string) string {
	quoted := strconv.Quote(value)
	return quoted[1 : len(quoted)-1]
}

func jsonPathValue(requestBody *matcher.BodyContext, expression string) (string, bool) {
	compiled, err := matcher.CompileJSONPath(expression)
	if err != nil {
		return "", false
	}
	parsed, err := requestBody.Parsed()
	if err != nil {
		return "", false
	}
	values := compiled.Values(parsed)
	if len(values) == 0 {
		return "", false
	}

	return stringify(values[0]), true
}

func stringify(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		encoded, err := json.Marshal(typed)
		if err == nil {
			return string(encoded)
		}
		return fmt.Sprint(typed)
	}
}

func cloneHeaders(source map[string][]string) http.Header {
	headers := make(http.Header, len(source))
	for name, values := range source {
		headers[name] = append([]string(nil), values...)
	}
	return headers
}
