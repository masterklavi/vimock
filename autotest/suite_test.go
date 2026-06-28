package autotest

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const (
	envBaseURL      = "VIMOCK_BASE_URL"
	envStartService = "VIMOCK_AUTOTEST_START"
	envBinary       = "VIMOCK_BINARY"
	envUpstreamHost = "VIMOCK_AUTOTEST_UPSTREAM_HOST"
)

var targetEnv *target

func TestMain(m *testing.M) {
	targetEnv = setupTarget()
	code := m.Run()
	if targetEnv != nil && targetEnv.cleanup != nil {
		targetEnv.cleanup()
	}
	os.Exit(code)
}

type target struct {
	enabled bool
	baseURL string
	client  *http.Client
	cleanup func()
	err     error
}

func setupTarget() *target {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv(envBaseURL)), "/")
	if baseURL != "" {
		return &target{
			enabled: true,
			baseURL: baseURL,
			client:  &http.Client{Timeout: 10 * time.Second},
			err:     waitForHealth(baseURL, 10*time.Second),
		}
	}

	if os.Getenv(envStartService) != "1" {
		return &target{client: &http.Client{Timeout: 10 * time.Second}}
	}

	return startLocalTarget()
}

func startLocalTarget() *target {
	binary := strings.TrimSpace(os.Getenv(envBinary))
	tempDir := ""
	if binary == "" {
		var err error
		tempDir, err = os.MkdirTemp("", "vimock-autotest-*")
		if err != nil {
			return &target{enabled: true, err: err}
		}
		binary = filepath.Join(tempDir, "vimock")
		build := exec.Command("go", "build", "-o", binary, "../cmd/vimock")
		build.Env = os.Environ()
		output, err := build.CombinedOutput()
		if err != nil {
			_ = os.RemoveAll(tempDir)
			return &target{enabled: true, err: fmt.Errorf("build vimock binary: %w\n%s", err, output)}
		}
	}

	port, err := freeTCPPort()
	if err != nil {
		if tempDir != "" {
			_ = os.RemoveAll(tempDir)
		}
		return &target{enabled: true, err: err}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, binary, "--host", "127.0.0.1", "--port", strconv.Itoa(port))
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		cancel()
		if tempDir != "" {
			_ = os.RemoveAll(tempDir)
		}
		return &target{enabled: true, err: fmt.Errorf("start vimock: %w", err)}
	}

	baseURL := "http://127.0.0.1:" + strconv.Itoa(port)
	cleanup := func() {
		cancel()
		if cmd.Process != nil {
			_ = cmd.Process.Signal(os.Interrupt)
		}
		done := make(chan struct{})
		go func() {
			_ = cmd.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			<-done
		}
		if tempDir != "" {
			_ = os.RemoveAll(tempDir)
		}
	}

	if err := waitForHealth(baseURL, 10*time.Second); err != nil {
		cleanup()
		return &target{enabled: true, err: fmt.Errorf("wait for vimock health: %w\n%s", err, output.String())}
	}

	return &target{
		enabled: true,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
		cleanup: cleanup,
	}
}

func freeTCPPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected listener addr %T", listener.Addr())
	}
	return addr.Port, nil
}

func waitForHealth(baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: time.Second}
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get(baseURL + "/__admin/health")
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = fmt.Errorf("health status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(100 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = errors.New("health timeout")
	}
	return lastErr
}

func requireTarget(t *testing.T) *target {
	t.Helper()
	if targetEnv == nil || !targetEnv.enabled {
		t.Skipf("black-box autotests are disabled; set %s or %s=1", envBaseURL, envStartService)
	}
	if targetEnv.err != nil {
		t.Fatalf("black-box target is not ready: %v", targetEnv.err)
	}
	return targetEnv
}

func (s *target) endpoint(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return s.baseURL + path
}

func (s *target) request(t *testing.T, method, path string, body []byte, headers map[string]string) (*http.Response, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, s.endpoint(path), reader)
	if err != nil {
		t.Fatalf("new request %s %s: %v", method, path, err)
	}
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response %s %s: %v", method, path, err)
	}
	return resp, responseBody
}

func (s *target) requestJSON(t *testing.T, method, path string, payload any) (*http.Response, []byte) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return s.request(t, method, path, body, map[string]string{"Content-Type": "application/json"})
}

func expectStatus(t *testing.T, resp *http.Response, body []byte, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Fatalf("status = %d, want %d: %s", resp.StatusCode, want, body)
	}
}

func decodeJSONBody[T any](t *testing.T, body []byte) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode JSON %q: %v", body, err)
	}
	return out
}

type mappingResponse struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Metadata map[string]any `json:"metadata"`
}

type mappingsListResponse struct {
	Mappings []mappingResponse `json:"mappings"`
	Meta     struct {
		Total int `json:"total"`
	} `json:"meta"`
}

func createMapping(t *testing.T, s *target, payload any) string {
	t.Helper()
	resp, body := s.requestJSON(t, http.MethodPost, "/__admin/mappings", payload)
	expectStatus(t, resp, body, http.StatusCreated)
	created := decodeJSONBody[mappingResponse](t, body)
	if created.ID == "" {
		t.Fatalf("created mapping id is empty: %s", body)
	}
	t.Cleanup(func() {
		deleteMappingIfExists(t, s, created.ID)
	})
	return created.ID
}

func createMappingRaw(t *testing.T, s *target, payload []byte) string {
	t.Helper()
	resp, body := s.request(t, http.MethodPost, "/__admin/mappings", payload, map[string]string{"Content-Type": "application/json"})
	expectStatus(t, resp, body, http.StatusCreated)
	created := decodeJSONBody[mappingResponse](t, body)
	if created.ID == "" {
		t.Fatalf("created mapping id is empty: %s", body)
	}
	t.Cleanup(func() {
		deleteMappingIfExists(t, s, created.ID)
	})
	return created.ID
}

func updateMapping(t *testing.T, s *target, id string, payload any) {
	t.Helper()
	resp, body := s.requestJSON(t, http.MethodPut, "/__admin/mappings/"+url.PathEscape(id), payload)
	expectStatus(t, resp, body, http.StatusOK)
}

func deleteMapping(t *testing.T, s *target, id string) {
	t.Helper()
	resp, body := s.request(t, http.MethodDelete, "/__admin/mappings/"+url.PathEscape(id), nil, nil)
	expectStatus(t, resp, body, http.StatusOK)
}

func deleteMappingIfExists(t *testing.T, s *target, id string) {
	t.Helper()
	resp, body := s.request(t, http.MethodDelete, "/__admin/mappings/"+url.PathEscape(id), nil, nil)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("cleanup delete status = %d, want 200/404: %s", resp.StatusCode, body)
	}
}

var uniqueCounter uint64

func uniqueName(t *testing.T) string {
	t.Helper()
	name := strings.ToLower(t.Name())
	name = strings.NewReplacer("/", "-", "_", "-", " ", "-", ".", "-").Replace(name)
	return fmt.Sprintf("%s-%d-%d", name, time.Now().UnixNano(), atomic.AddUint64(&uniqueCounter, 1))
}

func uploadLegacyFile(t *testing.T, s *target, fileName string, data []byte) {
	t.Helper()
	loginResp, loginBody := s.request(t, http.MethodPost, "/api/login", nil, nil)
	expectStatus(t, loginResp, loginBody, http.StatusOK)
	token := strings.TrimSpace(string(loginBody))
	metadata := "filename " + hex.EncodeToString([]byte(fileName))

	createResp, createBody := s.request(t, http.MethodPost, "/api/tus/"+url.PathEscape(fileName)+"?override=true", nil, map[string]string{
		"Tus-Resumable":   "1.0.0",
		"Upload-Length":   strconv.Itoa(len(data)),
		"Upload-Metadata": metadata,
		"X-Auth":          token,
	})
	expectStatus(t, createResp, createBody, http.StatusCreated)

	patchResp, patchBody := s.request(t, http.MethodPatch, "/api/tus/"+url.PathEscape(fileName)+"?override=true", data, map[string]string{
		"Content-Type":  "application/offset+octet-stream",
		"Tus-Resumable": "1.0.0",
		"Upload-Offset": "0",
		"X-Auth":        token,
	})
	expectStatus(t, patchResp, patchBody, http.StatusNoContent)
}

func newReachableUpstream(t *testing.T, handler http.Handler) (*httptest.Server, string) {
	t.Helper()
	server := httptest.NewUnstartedServer(handler)
	listenAddr := "127.0.0.1:0"
	if os.Getenv(envUpstreamHost) != "" {
		listenAddr = "0.0.0.0:0"
	}
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		t.Fatalf("listen upstream: %v", err)
	}
	server.Listener = listener
	server.Start()
	t.Cleanup(server.Close)

	upstreamURL := server.URL
	if host := strings.TrimSpace(os.Getenv(envUpstreamHost)); host != "" {
		parsed, err := url.Parse(server.URL)
		if err != nil {
			t.Fatalf("parse upstream URL: %v", err)
		}
		_, port, err := net.SplitHostPort(parsed.Host)
		if err != nil {
			t.Fatalf("split upstream host: %v", err)
		}
		parsed.Host = net.JoinHostPort(host, port)
		upstreamURL = parsed.String()
	}
	return server, upstreamURL
}
