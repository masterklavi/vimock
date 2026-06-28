package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestRunVersionAndInvalidConfig(t *testing.T) {
	var stdout bytes.Buffer
	code := run(context.Background(), []string{"--version"}, &stdout, testLogger(), nil, nil)
	if code != 0 || stdout.String() != "vimock dev\n" {
		t.Fatalf("version run code=%d stdout=%q", code, stdout.String())
	}

	code = run(context.Background(), []string{"--port", "0"}, io.Discard, testLogger(), nil, nil)
	if code != 2 {
		t.Fatalf("invalid config code=%d, want 2", code)
	}
}

func TestRunServerErrorAndGracefulShutdown(t *testing.T) {
	code := run(context.Background(), []string{"--host", "127.0.0.1", "--port", "18080"}, io.Discard, testLogger(), func(*http.Server) error {
		return errors.New("boom")
	}, nil)
	if code != 1 {
		t.Fatalf("server error code=%d, want 1", code)
	}

	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	codeCh := make(chan int, 1)
	go func() {
		codeCh <- run(ctx, []string{"--host", "127.0.0.1", "--port", "18081"}, io.Discard, testLogger(), func(*http.Server) error {
			close(started)
			<-ctx.Done()
			return http.ErrServerClosed
		}, nil)
	}()
	<-started
	cancel()
	select {
	case code := <-codeCh:
		if code != 0 {
			t.Fatalf("shutdown code=%d, want 0", code)
		}
	case <-time.After(time.Second):
		t.Fatal("run did not stop after context cancellation")
	}
}

func TestRunTLSConfigError(t *testing.T) {
	code := run(context.Background(), []string{
		"--host", "127.0.0.1",
		"--port", "18083",
		"--https-port", "18444",
		"--tls-cert-file", "missing.crt",
		"--tls-key-file", "missing.key",
	}, io.Discard, testLogger(), func(*http.Server) error {
		return http.ErrServerClosed
	}, nil)
	if code != 2 {
		t.Fatalf("TLS config error code=%d, want 2", code)
	}
}

func TestRunHTTPSStartsSecondListener(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	httpStarted := make(chan struct{})
	httpsStarted := make(chan struct{})
	codeCh := make(chan int, 1)
	go func() {
		codeCh <- run(ctx, []string{"--host", "127.0.0.1", "--port", "18082", "--https-port", "18443", "--tls-self-signed"}, io.Discard, testLogger(), func(*http.Server) error {
			close(httpStarted)
			<-ctx.Done()
			return http.ErrServerClosed
		}, func(server *http.Server) error {
			if server.TLSConfig == nil || len(server.TLSConfig.NextProtos) == 0 {
				return errors.New("missing TLS config")
			}
			close(httpsStarted)
			<-ctx.Done()
			return http.ErrServerClosed
		})
	}()
	<-httpStarted
	<-httpsStarted
	cancel()
	select {
	case code := <-codeCh:
		if code != 0 {
			t.Fatalf("https run code=%d, want 0", code)
		}
	case <-time.After(time.Second):
		t.Fatal("https run did not stop")
	}
}

func TestCertificateHosts(t *testing.T) {
	got := certificateHosts("127.0.0.1:8080")
	want := []string{"localhost", "127.0.0.1", "::1", "127.0.0.1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("certificateHosts = %#v, want %#v", got, want)
	}
}

func TestListenWrappersReturnErrors(t *testing.T) {
	if err := listenHTTP(&http.Server{Addr: "bad-address"}); err == nil {
		t.Fatal("listenHTTP invalid address error = nil")
	}
	if err := listenHTTPS(&http.Server{Addr: "bad-address"}); err == nil {
		t.Fatal("listenHTTPS invalid config error = nil")
	}
}

func TestShutdownServersJoinsErrors(t *testing.T) {
	server := &http.Server{}
	if err := shutdownServers([]*http.Server{server}); err != nil {
		t.Fatalf("shutdown unused server: %v", err)
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
