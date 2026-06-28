package server

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"vimock/internal/mapping"
	"vimock/internal/matcher"
)

const noMappingsMessage = "No response could be served as there are no stub mappings in this WireMock instance."

type runtimeAPI struct {
	mappings *mapping.Store
}

func (a runtimeAPI) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if isAdminPath(r.URL.Path) {
		http.NotFound(w, r)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		a.writeNoMatch(w, r)
		return
	}

	bodyContext := matcher.NewBodyContext(body)
	stub, ok := a.findMatch(r, bodyContext)
	if !ok {
		a.writeNoMatch(w, r)
		return
	}

	writeStubResponse(w, stub.Response())
}

func (a runtimeAPI) findMatch(r *http.Request, body *matcher.BodyContext) (mapping.Mapping, bool) {
	requestURI := r.URL.RequestURI()
	path := r.URL.Path
	query := r.URL.Query()
	headers := r.Header

	var selected mapping.Mapping
	var found bool
	for _, stub := range a.mappings.List() {
		if !stub.Request().Matches(r.Method, requestURI, path, query, headers, body) {
			continue
		}
		if !found || compareStubOrder(stub, selected) < 0 {
			selected = stub
			found = true
		}
	}

	return selected, found
}

func compareStubOrder(left, right mapping.Mapping) int {
	if left.Priority() != right.Priority() {
		return left.Priority() - right.Priority()
	}
	if left.Sequence() < right.Sequence() {
		return -1
	}
	if left.Sequence() > right.Sequence() {
		return 1
	}
	return 0
}

func (a runtimeAPI) writeNoMatch(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	if a.mappings.Count() == 0 {
		_, _ = w.Write([]byte(noMappingsMessage))
		return
	}

	_, _ = fmt.Fprint(w, "No response could be served as the request was not matched by any stub mapping.")
}

func writeStubResponse(w http.ResponseWriter, response mapping.ResponseDefinition) {
	for name, values := range response.Headers {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	if response.JSON && w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}

	w.WriteHeader(response.Status)
	if len(response.Body) > 0 {
		_, _ = w.Write(response.Body)
	}
}

func isAdminPath(path string) bool {
	return path == "/__admin" || strings.HasPrefix(path, "/__admin/")
}
