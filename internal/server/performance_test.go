package server

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"vimock/internal/mapping"
)

func BenchmarkRuntimeMatchAndRespondThousandMappings(b *testing.B) {
	store := largeMappingStore(b, 1000)
	handler := NewHandlerWithStore(slog.New(slog.NewTextHandler(io.Discard, nil)), store)

	b.ReportAllocs()
	b.ReportMetric(float64(store.Count()), "mappings")
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/item-999?source=bench", nil)
		resp := httptest.NewRecorder()

		handler.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			b.Fatalf("status = %d, want 200", resp.Code)
		}
		if got := resp.Body.String(); got != "ok-999" {
			b.Fatalf("body = %q, want ok-999", got)
		}
	}
}

func TestRuntimeLargeMappingSetSmoke(t *testing.T) {
	store := largeMappingStore(t, 5000)
	handler := NewHandlerWithStore(slog.New(slog.NewTextHandler(io.Discard, nil)), store)

	req := httptest.NewRequest(http.MethodGet, "/item-4999?source=bench", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.Code)
	}
	if got := resp.Body.String(); got != "ok-4999" {
		t.Fatalf("body = %q, want ok-4999", got)
	}
}

func largeMappingStore(tb testing.TB, count int) *mapping.Store {
	tb.Helper()

	store := mapping.NewStore()
	for i := 0; i < count; i++ {
		stub, err := mapping.ParseJSON([]byte(fmt.Sprintf(`{
		  "name": "bench-%d",
		  "request": {"method": "GET", "urlPath": "/item-%d", "queryParameters": {"source": {"equalTo": "bench"}}},
		  "response": {"status": 200, "body": "ok-%d"}
		}`, i, i, i)))
		if err != nil {
			tb.Fatalf("parse mapping %d: %v", i, err)
		}
		store.Create(stub)
	}
	return store
}
