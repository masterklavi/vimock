package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"vimock/internal/files"
	"vimock/internal/grpcdesc"
	"vimock/internal/mapping"
	"vimock/internal/proxy"
	"vimock/internal/recording"
	"vimock/internal/response"
	"vimock/internal/scenario"
)

type statusResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Service string `json:"service"`
}

func NewHandler(logger *slog.Logger) http.Handler {
	return NewHandlerWithStores(logger, mapping.NewStore(), files.NewMemoryStore())
}

func NewHandlerWithStore(logger *slog.Logger, mappings *mapping.Store) http.Handler {
	return NewHandlerWithStores(logger, mappings, files.NewMemoryStore())
}

func NewHandlerWithStores(logger *slog.Logger, mappings *mapping.Store, fileStore files.Store) http.Handler {
	return NewHandlerWithStoresAndDescriptors(logger, mappings, fileStore, grpcdesc.NewStore())
}

func NewHandlerWithStoresAndDescriptors(logger *slog.Logger, mappings *mapping.Store, fileStore files.Store, descriptorStore *grpcdesc.Store) http.Handler {
	return NewHandlerWithStoresDescriptorsRecorderForwarder(logger, mappings, fileStore, descriptorStore, recording.NewStore(), proxy.NewForwarder(&http.Client{
		Timeout: 30 * time.Second,
	}))
}

func NewHandlerWithStoresDescriptorsRecorderForwarder(logger *slog.Logger, mappings *mapping.Store, fileStore files.Store, descriptorStore *grpcdesc.Store, recorder *recording.Store, forwarder proxy.Forwarder) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	if mappings == nil {
		mappings = mapping.NewStore()
	}
	if fileStore == nil {
		fileStore = files.NewMemoryStore()
	}
	if descriptorStore == nil {
		descriptorStore = grpcdesc.NewStore()
	}
	if recorder == nil {
		recorder = recording.NewStore()
	}

	scenarios := scenario.NewStore()
	for _, stub := range mappings.List() {
		scenarios.MappingCreated(stub)
	}

	admin := adminAPI{
		mappings:    mappings,
		scenarios:   scenarios,
		descriptors: descriptorStore,
		recorder:    recorder,
	}
	filesAPI := fileAPI{
		files:       fileStore,
		descriptors: descriptorStore,
	}
	runtime := runtimeAPI{
		mappings:    mappings,
		descriptors: descriptorStore,
		renderer:    response.NewRenderer(fileStore),
		forwarder:   forwarder,
		scenarios:   scenarios,
		recorder:    recorder,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /__admin/health", healthHandler)
	mux.HandleFunc("GET /__admin/ready", readyHandler)
	mux.HandleFunc("GET /__admin/mappings", admin.listMappings)
	mux.HandleFunc("POST /__admin/mappings", admin.createMapping)
	mux.HandleFunc("GET /__admin/mappings/{id}", admin.getMapping)
	mux.HandleFunc("PUT /__admin/mappings/{id}", admin.updateMapping)
	mux.HandleFunc("DELETE /__admin/mappings/{id}", admin.deleteMapping)
	mux.HandleFunc("POST /__admin/scenarios/reset", admin.resetScenarios)
	mux.HandleFunc("POST /__admin/recordings/start", admin.startRecording)
	mux.HandleFunc("POST /__admin/recordings/stop", admin.stopRecording)
	mux.HandleFunc("POST /__admin/recordings/snapshot", admin.snapshotRecording)
	mux.HandleFunc("GET /__admin/ext/grpc/descriptors", admin.listGRPCDescriptors)
	mux.HandleFunc("PUT /__admin/ext/grpc/descriptors/{fileName}", admin.putGRPCDescriptor)
	mux.HandleFunc("DELETE /__admin/ext/grpc/descriptors/{fileName}", admin.deleteGRPCDescriptor)
	mux.HandleFunc("POST /__admin/ext/grpc/reset", admin.resetGRPC)
	mux.HandleFunc("POST /api/login", filesAPI.login)
	mux.HandleFunc("POST /api/tus/{file...}", filesAPI.createUpload)
	mux.HandleFunc("PATCH /api/tus/{file...}", filesAPI.patchUpload)
	mux.HandleFunc("/", runtime.serveHTTP)

	return loggingMiddleware(logger, mux)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, statusResponse{
		Status:  "healthy",
		Message: "VIMock is ok",
		Service: "vimock",
	})
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, statusResponse{
		Status:  "ready",
		Message: "VIMock is ready",
		Service: "vimock",
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		bodyCapture := wrapRequestBodyForLogging(r)

		next.ServeHTTP(recorder, r)

		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"duration_ms", time.Since(start).Milliseconds(),
		}
		if bodyCapture != nil {
			bodyCapture.ensureSample()
			attrs = append(attrs, bodyCapture.logAttrs(r.Header.Get("Content-Type"))...)
		}

		logger.Info("http request", attrs...)
	})
}

const maxLoggedRequestBodySize = 64 << 10

type requestBodyLogCapture struct {
	body      io.ReadCloser
	buffer    bytes.Buffer
	truncated bool
	readErr   error
	eof       bool
	closed    bool
}

func wrapRequestBodyForLogging(r *http.Request) *requestBodyLogCapture {
	if r.Body == nil || r.Body == http.NoBody {
		return nil
	}

	capture := &requestBodyLogCapture{body: r.Body}
	r.Body = capture
	return capture
}

func (c *requestBodyLogCapture) Read(p []byte) (int, error) {
	n, err := c.body.Read(p)
	if n > 0 {
		c.capture(p[:n])
	}
	if err == io.EOF {
		c.eof = true
	}
	if err != nil && err != io.EOF {
		c.readErr = err
	}
	return n, err
}

func (c *requestBodyLogCapture) Close() error {
	c.closed = true
	return c.body.Close()
}

func (c *requestBodyLogCapture) ensureSample() {
	if c == nil || c.truncated || c.eof || c.closed {
		return
	}

	buffer := make([]byte, 8<<10)
	for c.buffer.Len() < maxLoggedRequestBodySize {
		n, err := c.body.Read(buffer)
		if n > 0 {
			c.capture(buffer[:n])
		}
		if err != nil {
			if err != io.EOF {
				c.readErr = err
			}
			return
		}
		if n == 0 {
			return
		}
	}
}

func (c *requestBodyLogCapture) capture(chunk []byte) {
	remaining := maxLoggedRequestBodySize - c.buffer.Len()
	if remaining <= 0 {
		c.truncated = true
		return
	}
	if len(chunk) > remaining {
		_, _ = c.buffer.Write(chunk[:remaining])
		c.truncated = true
		return
	}
	_, _ = c.buffer.Write(chunk)
}

func (c *requestBodyLogCapture) logAttrs(contentType string) []any {
	if c == nil {
		return nil
	}

	attrs := make([]any, 0, 8)
	if c.readErr != nil {
		attrs = append(attrs, "request_body_read_error", c.readErr.Error())
	}
	if c.buffer.Len() == 0 {
		return attrs
	}

	body := c.buffer.Bytes()
	attrs = append(attrs, "request_body_truncated", c.truncated)
	if isTextLogBody(body, contentType) {
		attrs = append(attrs, "request_body", string(body))
		return attrs
	}

	attrs = append(attrs,
		"request_body_binary", true,
		"request_body_sample_bytes", len(body),
	)
	return attrs
}

func isTextLogBody(body []byte, contentType string) bool {
	if !utf8.Valid(body) || bytes.ContainsRune(body, '\x00') {
		return false
	}

	contentType = strings.ToLower(contentType)
	if strings.Contains(contentType, "grpc") ||
		strings.Contains(contentType, "octet-stream") {
		return false
	}

	return true
}
