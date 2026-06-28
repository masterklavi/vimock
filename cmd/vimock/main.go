package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vimock/internal/config"
	"vimock/internal/server"
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

	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetHTTP2(true)
	protocols.SetUnencryptedHTTP2(true)

	httpServer := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           server.NewHandler(logger),
		ReadHeaderTimeout: 5 * time.Second,
		Protocols:         protocols,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("starting vimock", "addr", cfg.Addr(), "protocols", protocols.String())
		errCh <- httpServer.ListenAndServe()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown failed", "error", err)
			os.Exit(1)
		}
		logger.Info("vimock stopped")
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}
}
