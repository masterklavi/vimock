package matcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type BodyPattern struct {
	jsonPath    *JSONPath
	absent      bool
	equalToJSON []byte
}

type BodyContext struct {
	raw        []byte
	parsed     any
	parsedErr  error
	parsedOnce bool
}

type EqualTo struct {
	Expected string
}

func NewBodyContext(raw []byte) *BodyContext {
	return &BodyContext{raw: raw}
}

func ParseBodyPattern(raw json.RawMessage) (BodyPattern, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return BodyPattern{}, fmt.Errorf("bodyPatterns entry must be a JSON object")
	}

	if rawJSONPath, ok := object["matchesJsonPath"]; ok {
		expression, absent, err := parseMatchesJSONPath(rawJSONPath)
		if err != nil {
			return BodyPattern{}, err
		}
		compiled, err := CompileJSONPath(expression)
		if err != nil {
			return BodyPattern{}, err
		}
		return BodyPattern{
			jsonPath: &compiled,
			absent:   absent,
		}, nil
	}

	if rawEqualToJSON, ok := object["equalToJson"]; ok {
		expected, err := normalizeRawJSON(rawEqualToJSON)
		if err != nil {
			return BodyPattern{}, fmt.Errorf("equalToJson must contain valid JSON: %w", err)
		}
		return BodyPattern{equalToJSON: expected}, nil
	}

	return BodyPattern{}, fmt.Errorf("unsupported bodyPatterns entry")
}

func ParseEqualToMap(raw json.RawMessage, field string) (map[string]EqualTo, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}

	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return nil, fmt.Errorf("%s must be a JSON object", field)
	}

	result := make(map[string]EqualTo, len(object))
	for name, rawMatcher := range object {
		var matcherObject map[string]json.RawMessage
		if err := json.Unmarshal(rawMatcher, &matcherObject); err != nil || matcherObject == nil {
			return nil, fmt.Errorf("%s.%s must be a JSON object", field, name)
		}

		rawEqualTo, ok := matcherObject["equalTo"]
		if !ok {
			return nil, fmt.Errorf("%s.%s must contain equalTo", field, name)
		}

		var expected string
		if err := json.Unmarshal(rawEqualTo, &expected); err != nil {
			return nil, fmt.Errorf("%s.%s.equalTo must be a string", field, name)
		}
		result[name] = EqualTo{Expected: expected}
	}

	return result, nil
}

func (p BodyPattern) Matches(body *BodyContext) bool {
	switch {
	case p.jsonPath != nil:
		parsedBody, err := body.Parsed()
		if err != nil {
			return false
		}
		exists := p.jsonPath.Exists(parsedBody)
		if p.absent {
			return !exists
		}
		return exists
	case p.equalToJSON != nil:
		actual, err := normalizeRawJSON(body.Raw())
		return err == nil && bytes.Equal(actual, p.equalToJSON)
	default:
		return true
	}
}

func MatchBodyPatterns(patterns []BodyPattern, body []byte) bool {
	return MatchBodyPatternsWithContext(patterns, NewBodyContext(body))
}

func MatchBodyPatternsWithContext(patterns []BodyPattern, body *BodyContext) bool {
	if len(patterns) == 0 {
		return true
	}

	for _, pattern := range patterns {
		if !pattern.Matches(body) {
			return false
		}
	}
	return true
}

func (c *BodyContext) Raw() []byte {
	if c == nil {
		return nil
	}
	return c.raw
}

func (c *BodyContext) Parsed() (any, error) {
	if c == nil {
		return nil, fmt.Errorf("request body is empty")
	}
	if !c.parsedOnce {
		c.parsed, c.parsedErr = ParseJSON(c.raw)
		c.parsedOnce = true
	}
	return c.parsed, c.parsedErr
}

func MatchQuery(patterns map[string]EqualTo, query url.Values) bool {
	for name, pattern := range patterns {
		values, ok := query[name]
		if !ok {
			return false
		}
		if !contains(values, pattern.Expected) {
			return false
		}
	}
	return true
}

func MatchHeaders(patterns map[string]EqualTo, headers http.Header) bool {
	for name, pattern := range patterns {
		values, ok := headers[http.CanonicalHeaderKey(name)]
		if !ok {
			values, ok = headers[name]
		}
		if !ok {
			return false
		}
		if !contains(values, pattern.Expected) {
			return false
		}
	}
	return true
}

func parseMatchesJSONPath(raw json.RawMessage) (string, bool, error) {
	var expression string
	if err := json.Unmarshal(raw, &expression); err == nil {
		return expression, false, nil
	}

	var object struct {
		Expression string `json:"expression"`
		Absent     bool   `json:"absent"`
	}
	if err := json.Unmarshal(raw, &object); err != nil {
		return "", false, fmt.Errorf("matchesJsonPath must be a string or object")
	}
	if object.Expression == "" {
		return "", false, fmt.Errorf("matchesJsonPath.expression must not be empty")
	}
	return object.Expression, object.Absent, nil
}

func normalizeRawJSON(raw []byte) ([]byte, error) {
	var jsonString string
	if err := json.Unmarshal(raw, &jsonString); err == nil {
		raw = []byte(jsonString)
	}

	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return json.Marshal(normalizeValue(value))
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
