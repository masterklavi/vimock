package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vimock/internal/config"
	"vimock/internal/server"
	"vimock/internal/tlsconfig"
)

var version = "dev"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	os.Exit(run(ctx, os.Args[1:], os.Stdout, logger, listenHTTP, listenHTTPS))
}

type listenFunc func(*http.Server) error

func run(ctx context.Context, args []string, stdout io.Writer, logger *slog.Logger, httpListen, httpsListen listenFunc) int {
	if stdout == nil {
		stdout = io.Discard
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if httpListen == nil {
		httpListen = listenHTTP
	}
	if httpsListen == nil {
		httpsListen = listenHTTPS
	}

	cfg, err := config.Load(args)
	if err != nil {
		logger.Error("invalid config", "error", err)
		return 2
	}
	if cfg.Version {
		_, _ = fmt.Fprintf(stdout, "vimock %s\n", version)
		return 0
	}

	handler := server.NewHandler(logger)
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetHTTP2(true)
	protocols.SetUnencryptedHTTP2(true)

	httpServer := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		Protocols:         protocols,
	}
	servers := []*http.Server{httpServer}

	if cfg.HTTPSEnabled() {
		tlsCfg, err := tlsconfig.Load(
			cfg.TLSCertFile,
			cfg.TLSKeyFile,
			cfg.TLSSelfSigned,
			certificateHosts(cfg.Host),
		)
		if err != nil {
			logger.Error("invalid tls config", "error", err)
			return 2
		}

		httpsProtocols := new(http.Protocols)
		httpsProtocols.SetHTTP1(true)
		httpsProtocols.SetHTTP2(true)
		httpsServer := &http.Server{
			Addr:              cfg.HTTPSAddr(),
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
			Protocols:         httpsProtocols,
			TLSConfig:         tlsCfg,
		}
		servers = append(servers, httpsServer)
	}

	errCh := make(chan serverError, len(servers))
	go func() {
		logger.Info("starting vimock", "addr", cfg.Addr(), "protocols", protocols.String())
		errCh <- serverError{name: "http", err: httpListen(httpServer)}
	}()
	if len(servers) > 1 {
		httpsServer := servers[1]
		go func() {
			logger.Info("starting vimock https", "addr", cfg.HTTPSAddr(), "protocols", httpsServer.Protocols.String())
			errCh <- serverError{name: "https", err: httpsListen(httpsServer)}
		}()
	}

	select {
	case <-ctx.Done():
		if err := shutdownServers(servers); err != nil {
			logger.Error("shutdown failed", "error", err)
			return 1
		}
		logger.Info("vimock stopped")
	case serverErr := <-errCh:
		if !errors.Is(serverErr.err, http.ErrServerClosed) {
			logger.Error("server failed", "listener", serverErr.name, "error", serverErr.err)
			_ = shutdownServers(servers)
			return 1
		}
	}
	return 0
}

type serverError struct {
	name string
	err  error
}

func shutdownServers(servers []*http.Server) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var result error
	for _, srv := range servers {
		if err := srv.Shutdown(shutdownCtx); err != nil {
			result = errors.Join(result, err)
		}
	}
	return result
}

func certificateHosts(host string) []string {
	hosts := []string{"localhost", "127.0.0.1", "::1"}
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	return append(hosts, host)
}

func listenHTTP(server *http.Server) error {
	return server.ListenAndServe()
}

func listenHTTPS(server *http.Server) error {
	return server.ListenAndServeTLS("", "")
}
