package mapping

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"vimock/internal/matcher"
)

const DefaultPriority = 5

// Mapping is a WireMock stub mapping with parsed top-level fields and preserved raw JSON.
type Mapping struct {
	id         string
	name       string
	persistent bool
	priority   int
	request    RequestPattern
	response   ResponseDefinition
	raw        map[string]json.RawMessage
	sequence   uint64
}

type RequestPattern struct {
	Method     string
	URL        string
	URLPath    string
	URLPattern string

	BodyPatterns    []matcher.BodyPattern
	QueryParameters map[string]matcher.EqualTo
	Headers         map[string]matcher.EqualTo
	urlRegex        *regexp.Regexp
}

type ResponseDefinition struct {
	Status                 int
	Headers                map[string][]string
	Body                   []byte
	JSON                   bool
	BodyFile               string
	Transformers           []string
	ProxyBaseURL           string
	ProxyURLPrefixToRemove string
	FixedDelayMilliseconds int
	DelayDistribution      *DelayDistribution
	ChunkedDribbleDelay    *ChunkedDribbleDelay
}

type DelayDistribution struct {
	Type   string
	Lower  int
	Upper  int
	Median int
	Sigma  float64
}

type ChunkedDribbleDelay struct {
	NumberOfChunks            int
	TotalDurationMilliseconds int
}

func ParseJSON(data []byte) (Mapping, error) {
	return ParseJSONWithID(data, "")
}

func ParseJSONWithID(data []byte, overrideID string) (Mapping, error) {
	raw, err := decodeObject(data)
	if err != nil {
		return Mapping{}, err
	}

	id, err := stringField(raw, "id")
	if err != nil {
		return Mapping{}, err
	}
	if id != "" {
		if err := ValidateID(id); err != nil {
			return Mapping{}, fmt.Errorf("id: %w", err)
		}
	}

	if overrideID != "" {
		if err := ValidateID(overrideID); err != nil {
			return Mapping{}, err
		}
		id = overrideID
	}
	if id == "" {
		id, err = NewID()
		if err != nil {
			return Mapping{}, fmt.Errorf("generate id: %w", err)
		}
	}

	name, err := stringField(raw, "name")
	if err != nil {
		return Mapping{}, err
	}
	persistent, err := boolField(raw, "persistent")
	if err != nil {
		return Mapping{}, err
	}
	priority, err := intField(raw, "priority", DefaultPriority)
	if err != nil {
		return Mapping{}, err
	}
	if err := validateObjectField(raw, "request", true); err != nil {
		return Mapping{}, err
	}
	if err := validateObjectField(raw, "response", true); err != nil {
		return Mapping{}, err
	}
	if err := validateObjectField(raw, "metadata", false); err != nil {
		return Mapping{}, err
	}
	request, err := parseRequest(raw["request"])
	if err != nil {
		return Mapping{}, err
	}
	response, err := parseResponse(raw["response"])
	if err != nil {
		return Mapping{}, err
	}

	mapping := Mapping{
		id:         id,
		name:       name,
		persistent: persistent,
		priority:   priority,
		request:    request,
		response:   response,
		raw:        cloneRawMap(raw),
	}
	mapping.raw["id"] = mustMarshalRaw(id)

	return mapping, nil
}

func (m Mapping) ID() string {
	return m.id
}

func (m Mapping) Name() string {
	return m.name
}

func (m Mapping) Persistent() bool {
	return m.persistent
}

func (m Mapping) Sequence() uint64 {
	return m.sequence
}

func (m Mapping) Priority() int {
	return m.priority
}

func (m Mapping) Request() RequestPattern {
	return m.request
}

func (m Mapping) Response() ResponseDefinition {
	return ResponseDefinition{
		Status:                 m.response.Status,
		Headers:                cloneHeaders(m.response.Headers),
		Body:                   cloneBytes(m.response.Body),
		JSON:                   m.response.JSON,
		BodyFile:               m.response.BodyFile,
		Transformers:           append([]string(nil), m.response.Transformers...),
		ProxyBaseURL:           m.response.ProxyBaseURL,
		ProxyURLPrefixToRemove: m.response.ProxyURLPrefixToRemove,
		FixedDelayMilliseconds: m.response.FixedDelayMilliseconds,
		DelayDistribution:      cloneDelayDistribution(m.response.DelayDistribution),
		ChunkedDribbleDelay:    cloneChunkedDribbleDelay(m.response.ChunkedDribbleDelay),
	}
}

func (m Mapping) WithID(id string) (Mapping, error) {
	if err := ValidateID(id); err != nil {
		return Mapping{}, err
	}

	next := m
	next.id = id
	next.raw = cloneRawMap(m.raw)
	next.raw["id"] = mustMarshalRaw(id)
	return next, nil
}

func (m Mapping) MarshalJSON() ([]byte, error) {
	raw := cloneRawMap(m.raw)
	raw["id"] = mustMarshalRaw(m.id)
	return json.Marshal(raw)
}

func decodeObject(data []byte) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var raw map[string]json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("mapping body must be a valid JSON object: %w", err)
	}
	if raw == nil {
		return nil, fmt.Errorf("mapping body must be a JSON object")
	}
	if decoder.More() {
		return nil, fmt.Errorf("mapping body must contain a single JSON object")
	}

	var extra json.RawMessage
	if err := decoder.Decode(&extra); err == nil {
		return nil, fmt.Errorf("mapping body must contain a single JSON object")
	}

	return raw, nil
}

func stringField(raw map[string]json.RawMessage, field string) (string, error) {
	value, ok := raw[field]
	if !ok || len(value) == 0 || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return "", nil
	}

	var parsed string
	if err := json.Unmarshal(value, &parsed); err != nil {
		return "", fmt.Errorf("%s must be a string", field)
	}
	return parsed, nil
}

func boolField(raw map[string]json.RawMessage, field string) (bool, error) {
	value, ok := raw[field]
	if !ok || len(value) == 0 || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return false, nil
	}

	var parsed bool
	if err := json.Unmarshal(value, &parsed); err != nil {
		return false, fmt.Errorf("%s must be a boolean", field)
	}
	return parsed, nil
}

func intField(raw map[string]json.RawMessage, field string, fallback int) (int, error) {
	value, ok := raw[field]
	if !ok || len(value) == 0 || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return fallback, nil
	}

	var parsed int
	if err := json.Unmarshal(value, &parsed); err != nil {
		return 0, fmt.Errorf("%s must be an integer", field)
	}
	return parsed, nil
}

func floatField(raw map[string]json.RawMessage, field string, fallback float64) (float64, error) {
	value, ok := raw[field]
	if !ok || len(value) == 0 || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return fallback, nil
	}

	var parsed float64
	if err := json.Unmarshal(value, &parsed); err != nil {
		return 0, fmt.Errorf("%s must be a number", field)
	}
	return parsed, nil
}

func validateObjectField(raw map[string]json.RawMessage, field string, required bool) error {
	value, ok := raw[field]
	if !ok {
		if required {
			return fmt.Errorf("%s is required", field)
		}
		return nil
	}

	var object map[string]json.RawMessage
	if err := json.Unmarshal(value, &object); err != nil || object == nil {
		return fmt.Errorf("%s must be a JSON object", field)
	}
	return nil
}

func parseRequest(raw json.RawMessage) (RequestPattern, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return RequestPattern{}, fmt.Errorf("request must be a JSON object")
	}

	method, err := stringField(object, "method")
	if err != nil {
		return RequestPattern{}, err
	}
	url, err := stringField(object, "url")
	if err != nil {
		return RequestPattern{}, err
	}
	urlPath, err := stringField(object, "urlPath")
	if err != nil {
		return RequestPattern{}, err
	}
	urlPattern, err := stringField(object, "urlPattern")
	if err != nil {
		return RequestPattern{}, err
	}
	bodyPatterns, err := parseBodyPatterns(object["bodyPatterns"])
	if err != nil {
		return RequestPattern{}, err
	}
	queryParameters, err := matcher.ParseEqualToMap(object["queryParameters"], "request.queryParameters")
	if err != nil {
		return RequestPattern{}, err
	}
	headers, err := matcher.ParseEqualToMap(object["headers"], "request.headers")
	if err != nil {
		return RequestPattern{}, err
	}

	var urlRegex *regexp.Regexp
	if urlPattern != "" {
		urlRegex, err = regexp.Compile(urlPattern)
		if err != nil {
			return RequestPattern{}, fmt.Errorf("request.urlPattern must be a valid regexp: %w", err)
		}
	}

	return RequestPattern{
		Method:     strings.ToUpper(method),
		URL:        url,
		URLPath:    urlPath,
		URLPattern: urlPattern,
		BodyPatterns: append([]matcher.BodyPattern(nil),
			bodyPatterns...,
		),
		QueryParameters: queryParameters,
		Headers:         headers,
		urlRegex:        urlRegex,
	}, nil
}

func parseBodyPatterns(raw json.RawMessage) ([]matcher.BodyPattern, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}

	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("request.bodyPatterns must be a JSON array")
	}

	patterns := make([]matcher.BodyPattern, 0, len(entries))
	for _, entry := range entries {
		pattern, err := matcher.ParseBodyPattern(entry)
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, pattern)
	}
	return patterns, nil
}

func (p RequestPattern) Matches(method, requestURI, path string, query map[string][]string, headers map[string][]string, body *matcher.BodyContext) bool {
	if !p.matchesMethod(method) {
		return false
	}
	if !p.matchesURL(requestURI, path) {
		return false
	}
	if !matcher.MatchQuery(p.QueryParameters, query) {
		return false
	}
	if !matcher.MatchHeaders(p.Headers, headers) {
		return false
	}
	if !matcher.MatchBodyPatternsWithContext(p.BodyPatterns, body) {
		return false
	}
	return true
}

func (p RequestPattern) matchesURL(requestURI, path string) bool {
	switch {
	case p.URL != "":
		return requestURI == p.URL
	case p.URLPath != "":
		return path == p.URLPath
	case p.urlRegex != nil:
		match := p.urlRegex.FindStringIndex(requestURI)
		return len(match) == 2 && match[0] == 0 && match[1] == len(requestURI)
	default:
		return false
	}
}

func (p RequestPattern) matchesMethod(method string) bool {
	switch p.Method {
	case "ANY":
		return true
	case "GET", "POST":
		return strings.EqualFold(p.Method, method)
	default:
		return false
	}
}

func parseResponse(raw json.RawMessage) (ResponseDefinition, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return ResponseDefinition{}, fmt.Errorf("response must be a JSON object")
	}

	status, err := intField(object, "status", httpStatusOK)
	if err != nil {
		return ResponseDefinition{}, err
	}
	if status < 100 || status > 999 {
		return ResponseDefinition{}, fmt.Errorf("response.status must be between 100 and 999")
	}

	headers, err := parseHeaders(object["headers"])
	if err != nil {
		return ResponseDefinition{}, err
	}

	body, isJSON, err := parseResponseBody(object)
	if err != nil {
		return ResponseDefinition{}, err
	}
	bodyFile, err := stringField(object, "bodyFileName")
	if err != nil {
		return ResponseDefinition{}, err
	}
	transformers, err := parseStringArrayField(object, "transformers")
	if err != nil {
		return ResponseDefinition{}, err
	}
	proxyBaseURL, err := stringField(object, "proxyBaseUrl")
	if err != nil {
		return ResponseDefinition{}, err
	}
	proxyURLPrefixToRemove, err := stringField(object, "proxyUrlPrefixToRemove")
	if err != nil {
		return ResponseDefinition{}, err
	}
	fixedDelayMilliseconds, err := intField(object, "fixedDelayMilliseconds", 0)
	if err != nil {
		return ResponseDefinition{}, err
	}
	if fixedDelayMilliseconds < 0 {
		return ResponseDefinition{}, fmt.Errorf("response.fixedDelayMilliseconds must be non-negative")
	}
	delayDistribution, err := parseDelayDistribution(object["delayDistribution"])
	if err != nil {
		return ResponseDefinition{}, err
	}
	chunkedDribbleDelay, err := parseChunkedDribbleDelay(object["chunkedDribbleDelay"])
	if err != nil {
		return ResponseDefinition{}, err
	}

	return ResponseDefinition{
		Status:                 status,
		Headers:                headers,
		Body:                   body,
		JSON:                   isJSON,
		BodyFile:               bodyFile,
		Transformers:           transformers,
		ProxyBaseURL:           proxyBaseURL,
		ProxyURLPrefixToRemove: proxyURLPrefixToRemove,
		FixedDelayMilliseconds: fixedDelayMilliseconds,
		DelayDistribution:      delayDistribution,
		ChunkedDribbleDelay:    chunkedDribbleDelay,
	}, nil
}

const httpStatusOK = 200

func parseHeaders(raw json.RawMessage) (map[string][]string, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}

	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return nil, fmt.Errorf("response.headers must be a JSON object")
	}

	headers := make(map[string][]string, len(object))
	for name, rawValue := range object {
		var value string
		if err := json.Unmarshal(rawValue, &value); err == nil {
			headers[name] = []string{value}
			continue
		}

		var values []string
		if err := json.Unmarshal(rawValue, &values); err == nil {
			headers[name] = values
			continue
		}

		return nil, fmt.Errorf("response.headers.%s must be a string or string array", name)
	}
	return headers, nil
}

func parseResponseBody(object map[string]json.RawMessage) ([]byte, bool, error) {
	if raw, ok := object["jsonBody"]; ok {
		return cloneBytes(raw), true, nil
	}

	raw, ok := object["body"]
	if !ok || len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, false, nil
	}

	var body string
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, false, fmt.Errorf("response.body must be a string")
	}
	return []byte(body), false, nil
}

func parseStringArrayField(raw map[string]json.RawMessage, field string) ([]string, error) {
	value, ok := raw[field]
	if !ok || len(value) == 0 || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return nil, nil
	}

	var parsed []string
	if err := json.Unmarshal(value, &parsed); err != nil {
		return nil, fmt.Errorf("%s must be a string array", field)
	}
	return parsed, nil
}

func parseDelayDistribution(raw json.RawMessage) (*DelayDistribution, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}

	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return nil, fmt.Errorf("response.delayDistribution must be a JSON object")
	}

	distributionType, err := stringField(object, "type")
	if err != nil {
		return nil, err
	}
	if distributionType == "" {
		return nil, fmt.Errorf("response.delayDistribution.type is required")
	}

	distribution := &DelayDistribution{Type: distributionType}
	switch distributionType {
	case "uniform":
		distribution.Lower, err = intField(object, "lower", 0)
		if err != nil {
			return nil, err
		}
		distribution.Upper, err = intField(object, "upper", 0)
		if err != nil {
			return nil, err
		}
		if distribution.Lower < 0 || distribution.Upper < distribution.Lower {
			return nil, fmt.Errorf("response.delayDistribution uniform bounds are invalid")
		}
	case "lognormal":
		distribution.Median, err = intField(object, "median", 0)
		if err != nil {
			return nil, err
		}
		distribution.Sigma, err = floatField(object, "sigma", 0)
		if err != nil {
			return nil, err
		}
		if distribution.Median <= 0 || distribution.Sigma < 0 {
			return nil, fmt.Errorf("response.delayDistribution lognormal parameters are invalid")
		}
	default:
		return nil, fmt.Errorf("response.delayDistribution.type %q is not supported", distributionType)
	}

	return distribution, nil
}

func parseChunkedDribbleDelay(raw json.RawMessage) (*ChunkedDribbleDelay, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}

	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return nil, fmt.Errorf("response.chunkedDribbleDelay must be a JSON object")
	}

	numberOfChunks, err := intField(object, "numberOfChunks", 0)
	if err != nil {
		return nil, err
	}
	totalDuration, err := intField(object, "totalDuration", 0)
	if err != nil {
		return nil, err
	}
	if numberOfChunks <= 0 || totalDuration < 0 {
		return nil, fmt.Errorf("response.chunkedDribbleDelay parameters are invalid")
	}

	return &ChunkedDribbleDelay{
		NumberOfChunks:            numberOfChunks,
		TotalDurationMilliseconds: totalDuration,
	}, nil
}

func cloneDelayDistribution(source *DelayDistribution) *DelayDistribution {
	if source == nil {
		return nil
	}
	clone := *source
	return &clone
}

func cloneChunkedDribbleDelay(source *ChunkedDribbleDelay) *ChunkedDribbleDelay {
	if source == nil {
		return nil
	}
	clone := *source
	return &clone
}

func cloneRawMap(source map[string]json.RawMessage) map[string]json.RawMessage {
	clone := make(map[string]json.RawMessage, len(source))
	for key, value := range source {
		clone[key] = cloneRaw(value)
	}
	return clone
}

func cloneRaw(source json.RawMessage) json.RawMessage {
	return cloneBytes(source)
}

func cloneBytes(source []byte) []byte {
	if source == nil {
		return nil
	}
	clone := make([]byte, len(source))
	copy(clone, source)
	return clone
}

func cloneHeaders(source map[string][]string) map[string][]string {
	if source == nil {
		return nil
	}
	clone := make(map[string][]string, len(source))
	for key, values := range source {
		clone[key] = append([]string(nil), values...)
	}
	return clone
}

func mustMarshalRaw(value any) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return raw
}

func NewID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}

	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	var out [36]byte
	hex.Encode(out[0:8], b[0:4])
	out[8] = '-'
	hex.Encode(out[9:13], b[4:6])
	out[13] = '-'
	hex.Encode(out[14:18], b[6:8])
	out[18] = '-'
	hex.Encode(out[19:23], b[8:10])
	out[23] = '-'
	hex.Encode(out[24:36], b[10:16])

	return string(out[:]), nil
}

func ValidateID(id string) error {
	if !IsValidID(id) {
		return fmt.Errorf("%s is not a valid UUID", id)
	}
	return nil
}

func IsValidID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i := 0; i < len(id); i++ {
		switch i {
		case 8, 13, 18, 23:
			if id[i] != '-' {
				return false
			}
		default:
			if !isHex(id[i]) {
				return false
			}
		}
	}
	return true
}

func isHex(b byte) bool {
	return (b >= '0' && b <= '9') ||
		(b >= 'a' && b <= 'f') ||
		(b >= 'A' && b <= 'F')
}
