package main

import (
	"context"
	"errors"
	"fmt"
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

	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		logger.Error("invalid config", "error", err)
		os.Exit(2)
	}
	if cfg.Version {
		fmt.Printf("vimock %s\n", version)
		return
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
			os.Exit(2)
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
		errCh <- serverError{name: "http", err: httpServer.ListenAndServe()}
	}()
	if len(servers) > 1 {
		httpsServer := servers[1]
		go func() {
			logger.Info("starting vimock https", "addr", cfg.HTTPSAddr(), "protocols", httpsServer.Protocols.String())
			errCh <- serverError{name: "https", err: httpsServer.ListenAndServeTLS("", "")}
		}()
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		if err := shutdownServers(servers); err != nil {
			logger.Error("shutdown failed", "error", err)
			os.Exit(1)
		}
		logger.Info("vimock stopped")
	case serverErr := <-errCh:
		if !errors.Is(serverErr.err, http.ErrServerClosed) {
			logger.Error("server failed", "listener", serverErr.name, "error", serverErr.err)
			_ = shutdownServers(servers)
			os.Exit(1)
		}
	}
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
