package recording

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"vimock/internal/mapping"
)

const (
	SourceStub      = "stub"
	SourceProxy     = "proxy"
	SourceRecording = "recording"

	ProtocolHTTP = "http"
	ProtocolGRPC = "grpc"
)

type Store struct {
	mu          sync.Mutex
	active      *session
	serveEvents []ServeEvent
	maxEvents   int
}

type session struct {
	spec   Spec
	events []ServeEvent
}

type Spec struct {
	TargetBaseURL       string                     `json:"targetBaseUrl,omitempty"`
	OutputFormat        string                     `json:"outputFormat,omitempty"`
	CaptureHeaders      map[string]json.RawMessage `json:"captureHeaders,omitempty"`
	RequestBodyPattern  string                     `json:"requestBodyPattern,omitempty"`
	RepeatsAsScenarios  bool                       `json:"repeatsAsScenarios,omitempty"`
	Persist             bool                       `json:"persist,omitempty"`
	ExtractBodyCriteria any                        `json:"extractBodyCriteria,omitempty"`
}

type ServeEvent struct {
	Method          string
	URL             string
	Path            string
	RawQuery        string
	RequestHeaders  http.Header
	RequestBody     []byte
	ResponseStatus  int
	ResponseHeaders http.Header
	ResponseBody    []byte
	Source          string
	Protocol        string
	RecordedAt      time.Time
}

type Snapshot struct {
	Mappings []mapping.Mapping `json:"mappings"`
	Meta     Meta              `json:"meta"`
}

type Meta struct {
	Total int `json:"total"`
}

func NewStore() *Store {
	return &Store{maxEvents: 1000}
}

func (s *Store) Start(spec Spec) error {
	if strings.TrimSpace(spec.TargetBaseURL) == "" {
		return fmt.Errorf("targetBaseUrl is required")
	}
	if err := validateTargetBaseURL(spec.TargetBaseURL); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.active = &session{spec: spec}
	return nil
}

func (s *Store) Stop() (Snapshot, error) {
	s.mu.Lock()
	active := s.active
	if active == nil {
		s.mu.Unlock()
		return Snapshot{}, fmt.Errorf("recording is not running")
	}
	s.active = nil
	events := cloneEvents(active.events)
	spec := active.spec
	s.mu.Unlock()

	return BuildSnapshot(events, spec)
}

func (s *Store) Snapshot(spec Spec) (Snapshot, error) {
	s.mu.Lock()
	events := cloneEvents(s.serveEvents)
	s.mu.Unlock()

	return BuildSnapshot(events, spec)
}

func (s *Store) ActiveSpec() (Spec, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active == nil {
		return Spec{}, false
	}
	return s.active.spec, true
}

func (s *Store) AddServeEvent(event ServeEvent) {
	s.addEvent(event, false)
}

func (s *Store) AddRecordingEvent(event ServeEvent) {
	s.addEvent(event, true)
}

func (s *Store) addEvent(event ServeEvent, recordingEvent bool) {
	if event.Protocol == "" {
		event.Protocol = ProtocolHTTP
	}
	if event.RecordedAt.IsZero() {
		event.RecordedAt = time.Now().UTC()
	}
	event.RequestHeaders = cloneHeaders(event.RequestHeaders)
	event.ResponseHeaders = cloneHeaders(event.ResponseHeaders)
	event.RequestBody = cloneBytes(event.RequestBody)
	event.ResponseBody = cloneBytes(event.ResponseBody)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.serveEvents = append(s.serveEvents, event)
	if s.maxEvents > 0 && len(s.serveEvents) > s.maxEvents {
		s.serveEvents = s.serveEvents[len(s.serveEvents)-s.maxEvents:]
	}
	if recordingEvent && s.active != nil {
		s.active.events = append(s.active.events, event)
	}
}

func BuildSnapshot(events []ServeEvent, spec Spec) (Snapshot, error) {
	mappings := make([]mapping.Mapping, 0, len(events))
	for _, event := range events {
		stub, err := mapping.ParseJSON(recordedMappingJSON(event, spec))
		if err != nil {
			return Snapshot{}, fmt.Errorf("build recorded mapping: %w", err)
		}
		mappings = append(mappings, stub)
	}
	return Snapshot{
		Mappings: mappings,
		Meta:     Meta{Total: len(mappings)},
	}, nil
}

func recordedMappingJSON(event ServeEvent, spec Spec) []byte {
	raw := map[string]any{
		"name":       recordedName(event),
		"persistent": spec.Persist,
		"request":    recordedRequest(event, spec),
		"response":   recordedResponse(event),
	}
	body, _ := json.Marshal(raw)
	return body
}

func recordedName(event ServeEvent) string {
	if event.URL != "" {
		return "Recorded " + event.Method + " " + event.URL
	}
	if event.RawQuery != "" {
		return "Recorded " + event.Method + " " + event.Path + "?" + event.RawQuery
	}
	return "Recorded " + event.Method + " " + event.Path
}

func recordedRequest(event ServeEvent, spec Spec) map[string]any {
	request := map[string]any{
		"method": event.Method,
	}
	if event.RawQuery != "" {
		request["url"] = event.Path + "?" + event.RawQuery
	} else if event.URL != "" {
		request["url"] = event.URL
	} else {
		request["urlPath"] = event.Path
	}

	if headers := capturedRequestHeaders(event.RequestHeaders, spec.CaptureHeaders); len(headers) > 0 {
		request["headers"] = headers
	}
	if patterns := requestBodyPatterns(event.RequestBody, spec); len(patterns) > 0 {
		request["bodyPatterns"] = patterns
	}
	return request
}

func capturedRequestHeaders(headers http.Header, capture map[string]json.RawMessage) map[string]any {
	if len(headers) == 0 || len(capture) == 0 {
		return nil
	}

	result := make(map[string]any)
	for name := range capture {
		if value := headerValue(headers, name); value != "" {
			result[name] = map[string]any{"equalTo": value}
		}
	}
	return result
}

func requestBodyPatterns(body []byte, spec Spec) []any {
	if len(body) == 0 {
		return nil
	}
	if spec.RequestBodyPattern != "" && spec.RequestBodyPattern != "equalToJson" {
		return nil
	}
	if !json.Valid(body) {
		return nil
	}
	return []any{
		map[string]any{
			"equalToJson": string(body),
		},
	}
}

func recordedResponse(event ServeEvent) map[string]any {
	response := map[string]any{
		"status": event.ResponseStatus,
	}
	if headers := recordedResponseHeaders(event); len(headers) > 0 {
		response["headers"] = headers
	}
	if len(event.ResponseBody) == 0 {
		return response
	}
	if isJSONResponse(event.ResponseHeaders, event.ResponseBody) {
		var body any
		if err := json.Unmarshal(event.ResponseBody, &body); err == nil {
			response["jsonBody"] = body
			return response
		}
	}
	if isTextBody(event.ResponseHeaders, event.ResponseBody) {
		response["body"] = string(event.ResponseBody)
		return response
	}

	response["base64Body"] = base64.StdEncoding.EncodeToString(event.ResponseBody)
	return response
}

func recordedResponseHeaders(event ServeEvent) map[string]any {
	headers := make(map[string]any)
	for name, values := range event.ResponseHeaders {
		if isExcludedResponseHeader(name) || len(values) == 0 {
			continue
		}
		if len(values) == 1 {
			headers[name] = values[0]
		} else {
			headers[name] = append([]string(nil), values...)
		}
	}
	if event.Protocol == ProtocolGRPC && headerValue(event.ResponseHeaders, "grpc-status-name") == "" {
		headers["grpc-status-name"] = "OK"
	}
	return headers
}

func isJSONResponse(headers http.Header, body []byte) bool {
	contentType := headers.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = contentType
	}
	return strings.Contains(strings.ToLower(mediaType), "json") && json.Valid(body)
}

func isTextBody(headers http.Header, body []byte) bool {
	contentType := strings.ToLower(headers.Get("Content-Type"))
	return utf8.Valid(body) && !strings.ContainsRune(string(body), 0) &&
		(contentType == "" || strings.HasPrefix(contentType, "text/") || strings.Contains(contentType, "json") || strings.Contains(contentType, "xml"))
}

func validateTargetBaseURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("targetBaseUrl is invalid: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("targetBaseUrl must include scheme and host")
	}
	return nil
}

func isExcludedResponseHeader(name string) bool {
	switch strings.ToLower(name) {
	case "content-length", "connection", "date", "transfer-encoding":
		return true
	default:
		return false
	}
}

func headerValue(headers http.Header, name string) string {
	if value := headers.Get(name); value != "" {
		return value
	}
	for candidate, values := range headers {
		if strings.EqualFold(candidate, name) && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

func cloneEvents(events []ServeEvent) []ServeEvent {
	clone := make([]ServeEvent, len(events))
	for i, event := range events {
		clone[i] = event
		clone[i].RequestHeaders = cloneHeaders(event.RequestHeaders)
		clone[i].ResponseHeaders = cloneHeaders(event.ResponseHeaders)
		clone[i].RequestBody = cloneBytes(event.RequestBody)
		clone[i].ResponseBody = cloneBytes(event.ResponseBody)
	}
	return clone
}

func cloneHeaders(source http.Header) http.Header {
	if source == nil {
		return nil
	}
	clone := make(http.Header, len(source))
	for name, values := range source {
		clone[name] = append([]string(nil), values...)
	}
	return clone
}

func cloneBytes(source []byte) []byte {
	if source == nil {
		return nil
	}
	clone := make([]byte, len(source))
	copy(clone, source)
	return clone
}
