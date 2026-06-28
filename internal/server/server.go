package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"vimock/internal/mapping"
)

type statusResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Service string `json:"service"`
}

func NewHandler(logger *slog.Logger) http.Handler {
	return NewHandlerWithStore(logger, mapping.NewStore())
}

func NewHandlerWithStore(logger *slog.Logger, mappings *mapping.Store) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	if mappings == nil {
		mappings = mapping.NewStore()
	}

	admin := adminAPI{mappings: mappings}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /__admin/health", healthHandler)
	mux.HandleFunc("GET /__admin/ready", readyHandler)
	mux.HandleFunc("GET /__admin/mappings", admin.listMappings)
	mux.HandleFunc("POST /__admin/mappings", admin.createMapping)
	mux.HandleFunc("GET /__admin/mappings/{id}", admin.getMapping)
	mux.HandleFunc("PUT /__admin/mappings/{id}", admin.updateMapping)
	mux.HandleFunc("DELETE /__admin/mappings/{id}", admin.deleteMapping)

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
