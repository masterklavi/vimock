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
		Headers: http.Header(definition.Headers),
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
		rendered.Body = definition.Body
		templatable = true
	}

	if templatable && definition.UsesResponseTemplate() && len(rendered.Body) > 0 {
		if definition.Template != nil {
			rendered.Body = renderCompiledTemplate(definition.Template, requestBody, definition.JSON)
		} else {
			rendered.Body = renderTemplate(rendered.Body, requestBody, definition.JSON)
		}
	}

	return rendered, nil
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

func renderCompiledTemplate(template *mapping.ResponseTemplate, requestBody *matcher.BodyContext, escapeJSONString bool) []byte {
	if template == nil {
		return nil
	}
	if requestBody == nil {
		return nil
	}
	parsed, err := requestBody.Parsed()
	if err != nil {
		return nil
	}

	size := 0
	for _, segment := range template.Segments {
		size += len(segment.Literal)
	}
	rendered := make([]byte, 0, size)
	for _, segment := range template.Segments {
		if !segment.Helper {
			rendered = append(rendered, segment.Literal...)
			continue
		}
		value, ok := segment.JSONPath.FirstValue(parsed)
		if !ok {
			continue
		}
		renderedValue := stringify(value)
		if escapeJSONString {
			renderedValue = escapeJSONStringContent(renderedValue)
		}
		rendered = append(rendered, renderedValue...)
	}
	return rendered
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
