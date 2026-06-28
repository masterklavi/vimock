package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"vimock/internal/delay"
	"vimock/internal/grpcdesc"
	"vimock/internal/mapping"
	"vimock/internal/matcher"
	"vimock/internal/proxy"
	"vimock/internal/recording"
	"vimock/internal/response"
	"vimock/internal/scenario"
)

const noMappingsMessage = "No response could be served as there are no stub mappings in this WireMock instance."

type runtimeAPI struct {
	mappings    *mapping.Store
	descriptors *grpcdesc.Store
	renderer    response.Renderer
	forwarder   proxy.Forwarder
	scenarios   scenario.StateStore
	sleeper     delay.Sleeper
	recorder    *recording.Store
}

func (a runtimeAPI) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if isAdminPath(r.URL.Path) {
		http.NotFound(w, r)
		return
	}
	if isGRPCRequest(r) {
		a.serveGRPC(w, r)
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
		if a.tryServeRecordingProxy(w, r, body) {
			return
		}
		a.writeNoMatch(w, r)
		return
	}

	responseDefinition := stub.Response()
	if err := a.sleep(r.Context(), delay.InitialDuration(responseDefinition, nil)); err != nil {
		return
	}

	if responseDefinition.ProxyBaseURL != "" {
		proxied, err := a.forwarder.Forward(r.Context(), r, body, responseDefinition)
		if err != nil {
			writeProxyError(w, err)
			return
		}
		a.recordServeEvent(r, body, proxied.Status, proxied.Headers, proxied.Body, recording.SourceProxy)
		writeProxyResponse(w, r, proxied, responseDefinition, a.sleep)
		return
	}

	rendered, err := a.renderer.Render(responseDefinition, bodyContext)
	if err != nil {
		writeResponseRenderError(w, err)
		return
	}

	a.recordServeEvent(r, body, rendered.Status, rendered.Headers, rendered.Body, recording.SourceStub)
	writeStubResponse(w, r, rendered, responseDefinition, a.sleep)
}

func (a runtimeAPI) tryServeRecordingProxy(w http.ResponseWriter, r *http.Request, body []byte) bool {
	if a.recorder == nil {
		return false
	}
	spec, ok := a.recorder.ActiveSpec()
	if !ok {
		return false
	}

	proxied, err := a.forwarder.Forward(r.Context(), r, body, mapping.ResponseDefinition{
		ProxyBaseURL: spec.TargetBaseURL,
	})
	if err != nil {
		writeProxyError(w, err)
		return true
	}

	event := newServeEvent(r, body, proxied.Status, proxied.Headers, proxied.Body, recording.SourceRecording)
	a.recorder.AddRecordingEvent(event)
	writeProxyResponse(w, r, proxied, mapping.ResponseDefinition{}, a.sleep)
	return true
}

func (a runtimeAPI) recordServeEvent(r *http.Request, requestBody []byte, responseStatus int, responseHeaders http.Header, responseBody []byte, source string) {
	if a.recorder == nil {
		return
	}
	a.recorder.AddServeEvent(newServeEvent(r, requestBody, responseStatus, responseHeaders, responseBody, source))
}

func newServeEvent(r *http.Request, requestBody []byte, responseStatus int, responseHeaders http.Header, responseBody []byte, source string) recording.ServeEvent {
	return recording.ServeEvent{
		Method:          r.Method,
		URL:             r.URL.RequestURI(),
		Path:            r.URL.Path,
		RawQuery:        r.URL.RawQuery,
		RequestHeaders:  r.Header,
		RequestBody:     requestBody,
		ResponseStatus:  responseStatus,
		ResponseHeaders: responseHeaders,
		ResponseBody:    responseBody,
		Source:          source,
		Protocol:        recording.ProtocolHTTP,
	}
}

func (a runtimeAPI) findMatch(r *http.Request, body *matcher.BodyContext) (mapping.Mapping, bool) {
	requestURI := r.URL.RequestURI()
	path := r.URL.Path
	query := r.URL.Query()
	headers := r.Header

	candidates := make([]mapping.Mapping, 0)
	for _, stub := range a.mappings.List() {
		if !stub.Request().Matches(r.Method, requestURI, path, query, headers, body) {
			continue
		}
		candidates = append(candidates, stub)
	}

	if a.scenarios == nil {
		return selectBestStub(candidates)
	}
	return a.scenarios.SelectAndTransition(candidates, compareStubOrder)
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

func selectBestStub(candidates []mapping.Mapping) (mapping.Mapping, bool) {
	var selected mapping.Mapping
	var found bool
	for _, stub := range candidates {
		if !found || compareStubOrder(stub, selected) < 0 {
			selected = stub
			found = true
		}
	}
	return selected, found
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

func writeResponseRenderError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func writeProxyError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusBadGateway)
}

func writeStubResponse(w http.ResponseWriter, r *http.Request, rendered response.Rendered, definition mapping.ResponseDefinition, sleeper delay.Sleeper) {
	for name, values := range rendered.Headers {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	if rendered.JSON && w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}

	w.WriteHeader(rendered.Status)
	_ = writeResponseBody(r, w, rendered.Body, definition, sleeper)
}

func writeProxyResponse(w http.ResponseWriter, r *http.Request, proxied proxy.Response, definition mapping.ResponseDefinition, sleeper delay.Sleeper) {
	for name, values := range proxied.Headers {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	if definition.ChunkedDribbleDelay != nil {
		w.Header().Del("Content-Length")
	}
	w.WriteHeader(proxied.Status)
	_ = writeResponseBody(r, w, proxied.Body, definition, sleeper)
}

func writeResponseBody(r *http.Request, w http.ResponseWriter, body []byte, definition mapping.ResponseDefinition, sleeper delay.Sleeper) error {
	if len(body) == 0 {
		return nil
	}

	chunks, interval := delay.ChunkedInterval(definition)
	if chunks <= 1 || len(body) == 1 {
		_, err := w.Write(body)
		return err
	}

	parts := splitBody(body, chunks)
	for index, chunk := range parts {
		if len(chunk) == 0 {
			continue
		}
		if _, err := w.Write(chunk); err != nil {
			return err
		}
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		if index < len(parts)-1 {
			if err := sleeper(r.Context(), interval); err != nil {
				return err
			}
		}
	}
	return nil
}

func splitBody(body []byte, chunks int) [][]byte {
	if chunks <= 1 || chunks >= len(body) {
		result := make([][]byte, 0, len(body))
		for index := range body {
			result = append(result, body[index:index+1])
		}
		return result
	}

	result := make([][]byte, 0, chunks)
	for index := 0; index < chunks; index++ {
		start := index * len(body) / chunks
		end := (index + 1) * len(body) / chunks
		result = append(result, body[start:end])
	}
	return result
}

func (a runtimeAPI) sleep(ctx context.Context, duration time.Duration) error {
	sleeper := a.sleeper
	if sleeper == nil {
		sleeper = delay.Sleep
	}
	return sleeper(ctx, duration)
}

func isAdminPath(path string) bool {
	return path == "/__admin" || strings.HasPrefix(path, "/__admin/")
}
