package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"vimock/internal/files"
	"vimock/internal/mapping"
	"vimock/internal/proxy"
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
	if logger == nil {
		logger = slog.Default()
	}
	if mappings == nil {
		mappings = mapping.NewStore()
	}
	if fileStore == nil {
		fileStore = files.NewMemoryStore()
	}

	scenarios := scenario.NewStore()
	for _, stub := range mappings.List() {
		scenarios.MappingCreated(stub)
	}

	admin := adminAPI{
		mappings:  mappings,
		scenarios: scenarios,
	}
	filesAPI := fileAPI{files: fileStore}
	runtime := runtimeAPI{
		mappings: mappings,
		renderer: response.NewRenderer(fileStore),
		forwarder: proxy.NewForwarder(&http.Client{
			Timeout: 30 * time.Second,
		}),
		scenarios: scenarios,
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
	mux.HandleFunc("POST /__admin/ext/grpc/reset", admin.resetGRPC)
	mux.HandleFunc("POST /api/login", filesAPI.login)
	mux.HandleFunc("POST /api/tus/{file}", filesAPI.createUpload)
	mux.HandleFunc("PATCH /api/tus/{file}", filesAPI.patchUpload)
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

		next.ServeHTTP(recorder, r)

		logger.Info(
			"http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
