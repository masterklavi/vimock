package matcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type JSONPath struct {
	expression string
	path       []pathToken
	filter     *filter
}

type pathToken struct {
	kind  tokenKind
	field string
	index int
}

type tokenKind int

const (
	tokenField tokenKind = iota
	tokenIndex
	tokenWildcard
)

type filter struct {
	path      []pathToken
	useSelf   bool
	sizeCheck bool
	expected  any
}

func CompileJSONPath(expression string) (JSONPath, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return JSONPath{}, fmt.Errorf("JSONPath expression must not be empty")
	}
	if !strings.HasPrefix(expression, "$") {
		return JSONPath{}, fmt.Errorf("JSONPath expression must start with $")
	}

	pathExpression, filterExpression, err := splitFilter(expression)
	if err != nil {
		return JSONPath{}, err
	}

	path, err := parsePath(pathExpression)
	if err != nil {
		return JSONPath{}, err
	}

	var parsedFilter *filter
	if filterExpression != "" {
		parsed, err := parseFilter(filterExpression)
		if err != nil {
			return JSONPath{}, err
		}
		parsedFilter = &parsed
	}

	return JSONPath{
		expression: expression,
		path:       path,
		filter:     parsedFilter,
	}, nil
}

func (p JSONPath) Exists(root any) bool {
	nodes := evalPath([]any{root}, p.path)
	if p.filter == nil {
		return len(nodes) > 0
	}

	for _, node := range nodes {
		for _, candidate := range filterCandidates(node) {
			if p.filter.matches(candidate) {
				return true
			}
		}
	}
	return false
}

func (p JSONPath) Values(root any) []any {
	nodes := evalPath([]any{root}, p.path)
	if p.filter == nil {
		return nodes
	}

	var values []any
	for _, node := range nodes {
		for _, candidate := range filterCandidates(node) {
			if p.filter.matches(candidate) {
				values = append(values, candidate)
			}
		}
	}
	return values
}

func ParseJSON(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func splitFilter(expression string) (string, string, error) {
	start := strings.Index(expression, "[?(")
	if start == -1 {
		return expression, "", nil
	}
	if !strings.HasSuffix(expression, ")]") {
		return "", "", fmt.Errorf("unsupported JSONPath filter syntax: %s", expression)
	}
	return expression[:start], expression[start+3 : len(expression)-2], nil
}

func parsePath(expression string) ([]pathToken, error) {
	if expression == "$" {
		return nil, nil
	}
	if !strings.HasPrefix(expression, "$") {
		return nil, fmt.Errorf("path must start with $: %s", expression)
	}

	var tokens []pathToken
	for i := 1; i < len(expression); {
		switch expression[i] {
		case '.':
			i++
			if i < len(expression) && expression[i] == '*' {
				tokens = append(tokens, pathToken{kind: tokenWildcard})
				i++
				continue
			}
			start := i
			for i < len(expression) && expression[i] != '.' && expression[i] != '[' {
				i++
			}
			if start == i {
				return nil, fmt.Errorf("empty field in JSONPath: %s", expression)
			}
			tokens = append(tokens, pathToken{kind: tokenField, field: expression[start:i]})
		case '[':
			end := strings.IndexByte(expression[i:], ']')
			if end == -1 {
				return nil, fmt.Errorf("unclosed index in JSONPath: %s", expression)
			}
			end += i
			index, err := strconv.Atoi(strings.TrimSpace(expression[i+1 : end]))
			if err != nil {
				return nil, fmt.Errorf("unsupported JSONPath index %q", expression[i+1:end])
			}
			tokens = append(tokens, pathToken{kind: tokenIndex, index: index})
			i = end + 1
		default:
			return nil, fmt.Errorf("unsupported JSONPath segment at %q in %s", expression[i:], expression)
		}
	}

	return tokens, nil
}

func parseFilter(expression string) (filter, error) {
	left, right, ok := strings.Cut(expression, "==")
	if !ok {
		return filter{}, fmt.Errorf("unsupported JSONPath filter condition: %s", expression)
	}

	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if !strings.HasPrefix(left, "@") {
		return filter{}, fmt.Errorf("JSONPath filter must start with @: %s", expression)
	}

	useSelf := left == "@"
	sizeCheck := strings.HasSuffix(left, ".size()")
	if sizeCheck {
		left = strings.TrimSuffix(left, ".size()")
	}

	var path []pathToken
	var err error
	if !useSelf {
		path, err = parsePath("$" + strings.TrimPrefix(left, "@"))
		if err != nil {
			return filter{}, err
		}
	}

	expected, err := parseLiteral(right)
	if err != nil {
		return filter{}, err
	}

	return filter{
		path:      path,
		useSelf:   useSelf,
		sizeCheck: sizeCheck,
		expected:  expected,
	}, nil
}

func parseLiteral(value string) (any, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		return value[1 : len(value)-1], nil
	}

	jsonValue := strings.ReplaceAll(value, "'", `"`)
	var parsed any
	decoder := json.NewDecoder(strings.NewReader(jsonValue))
	decoder.UseNumber()
	if err := decoder.Decode(&parsed); err != nil {
		return nil, fmt.Errorf("unsupported JSONPath literal %q: %w", value, err)
	}
	return parsed, nil
}

func evalPath(nodes []any, tokens []pathToken) []any {
	current := nodes
	for _, token := range tokens {
		var next []any
		for _, node := range current {
			switch token.kind {
			case tokenField:
				if object, ok := node.(map[string]any); ok {
					if value, exists := object[token.field]; exists {
						next = append(next, value)
					}
				}
			case tokenIndex:
				if array, ok := node.([]any); ok && token.index >= 0 && token.index < len(array) {
					next = append(next, array[token.index])
				}
			case tokenWildcard:
				switch value := node.(type) {
				case []any:
					next = append(next, value...)
				case map[string]any:
					for _, item := range value {
						next = append(next, item)
					}
				}
			}
		}
		current = next
		if len(current) == 0 {
			return nil
		}
	}
	return current
}

func filterCandidates(node any) []any {
	if array, ok := node.([]any); ok {
		return array
	}
	return []any{node}
}

func (f filter) matches(candidate any) bool {
	value := candidate
	if !f.useSelf {
		values := evalPath([]any{candidate}, f.path)
		if len(values) == 0 {
			return false
		}
		value = values[0]
	}

	if f.sizeCheck {
		size, ok := valueSize(value)
		return ok && compareValues(json.Number(strconv.Itoa(size)), f.expected)
	}

	return compareValues(value, f.expected)
}

func valueSize(value any) (int, bool) {
	switch typed := value.(type) {
	case []any:
		return len(typed), true
	case map[string]any:
		return len(typed), true
	case string:
		return len(typed), true
	default:
		return 0, false
	}
}

func compareValues(left, right any) bool {
	left = normalizeValue(left)
	right = normalizeValue(right)
	return reflect.DeepEqual(left, right)
}

func normalizeValue(value any) any {
	switch typed := value.(type) {
	case json.Number:
		if strings.ContainsAny(typed.String(), ".eE") {
			if f, err := typed.Float64(); err == nil {
				return f
			}
		}
		if i, err := typed.Int64(); err == nil {
			return i
		}
		return typed.String()
	case float64:
		if typed == float64(int64(typed)) {
			return int64(typed)
		}
		return typed
	case []any:
		normalized := make([]any, len(typed))
		for i, item := range typed {
			normalized[i] = normalizeValue(item)
		}
		return normalized
	case map[string]any:
		normalized := make(map[string]any, len(typed))
		for key, item := range typed {
			normalized[key] = normalizeValue(item)
		}
		return normalized
	default:
		return typed
	}
}
