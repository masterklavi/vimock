package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"vimock/internal/mapping"
)

type Forwarder struct {
	client *http.Client
}

type Response struct {
	Status  int
	Headers http.Header
	Body    []byte
}

func NewForwarder(client *http.Client) Forwarder {
	if client == nil {
		client = http.DefaultClient
	}
	return Forwarder{client: client}
}

func (f Forwarder) Forward(ctx context.Context, original *http.Request, body []byte, definition mapping.ResponseDefinition) (Response, error) {
	target, err := TargetURL(definition.ProxyBaseURL, definition.ProxyURLPrefixToRemove, original.URL)
	if err != nil {
		return Response{}, err
	}

	req, err := http.NewRequestWithContext(ctx, original.Method, target, bytes.NewReader(body))
	if err != nil {
		return Response{}, fmt.Errorf("create proxy request: %w", err)
	}
	copyHeaders(req.Header, original.Header)

	client := f.client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("proxy request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, fmt.Errorf("read proxy response: %w", err)
	}

	return Response{
		Status:  resp.StatusCode,
		Headers: cloneHeaders(resp.Header),
		Body:    responseBody,
	}, nil
}

func TargetURL(baseURL, prefixToRemove string, original *url.URL) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse proxyBaseUrl: %w", err)
	}
	if base.Scheme == "" || base.Host == "" {
		return "", fmt.Errorf("proxyBaseUrl must include scheme and host")
	}

	path := original.Path
	if prefixToRemove != "" && strings.HasPrefix(path, prefixToRemove) {
		path = strings.TrimPrefix(path, prefixToRemove)
		if path == "" {
			path = "/"
		}
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	target := *base
	target.Path = joinURLPath(base.Path, path)
	target.RawQuery = joinRawQuery(base.RawQuery, original.RawQuery)
	target.Fragment = ""
	return target.String(), nil
}

func copyHeaders(target, source http.Header) {
	for name, values := range source {
		if isHopByHopHeader(name) {
			continue
		}
		for _, value := range values {
			target.Add(name, value)
		}
	}
}

func cloneHeaders(source http.Header) http.Header {
	headers := make(http.Header, len(source))
	for name, values := range source {
		if isHopByHopHeader(name) {
			continue
		}
		headers[name] = append([]string(nil), values...)
	}
	return headers
}

func isHopByHopHeader(name string) bool {
	switch strings.ToLower(name) {
	case "connection",
		"keep-alive",
		"proxy-authenticate",
		"proxy-authorization",
		"proxy-connection",
		"te",
		"trailer",
		"transfer-encoding",
		"upgrade":
		return true
	default:
		return false
	}
}

func joinURLPath(basePath, requestPath string) string {
	switch {
	case basePath == "" || basePath == "/":
		return requestPath
	case requestPath == "" || requestPath == "/":
		return basePath
	case strings.HasSuffix(basePath, "/") && strings.HasPrefix(requestPath, "/"):
		return basePath + strings.TrimPrefix(requestPath, "/")
	case !strings.HasSuffix(basePath, "/") && !strings.HasPrefix(requestPath, "/"):
		return basePath + "/" + requestPath
	default:
		return basePath + requestPath
	}
}

func joinRawQuery(baseQuery, requestQuery string) string {
	switch {
	case baseQuery == "":
		return requestQuery
	case requestQuery == "":
		return baseQuery
	default:
		return baseQuery + "&" + requestQuery
	}
}
