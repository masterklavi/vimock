package config

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	defaultHost = "0.0.0.0"
	defaultPort = 8080
)

// Config contains the process-level settings needed by the bootstrap server.
type Config struct {
	Host string
	Port int
}

func (c Config) Addr() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

// Load parses configuration from environment and command-line flags.
// Flags intentionally override env vars so local runs can adjust settings without mutating shell state.
func Load(args []string) (Config, error) {
	return load(args, os.Getenv, io.Discard)
}

func load(args []string, getenv func(string) string, output io.Writer) (Config, error) {
	if output == nil {
		output = io.Discard
	}

	cfg := Config{
		Host: stringFromEnv(getenv, "VIMOCK_HOST", defaultHost),
		Port: intFromEnv(getenv, "VIMOCK_PORT", defaultPort),
	}

	fs := flag.NewFlagSet("vimock", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.StringVar(&cfg.Host, "host", cfg.Host, "bind host, overrides VIMOCK_HOST")
	fs.IntVar(&cfg.Port, "port", cfg.Port, "bind port, overrides VIMOCK_PORT")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if cfg.Host = strings.TrimSpace(cfg.Host); cfg.Host == "" {
		return Config{}, fmt.Errorf("host must not be empty")
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return Config{}, fmt.Errorf("port must be between 1 and 65535")
	}

	return cfg, nil
}

func stringFromEnv(getenv func(string) string, key, fallback string) string {
	if value := strings.TrimSpace(getenv(key)); value != "" {
		return value
	}
	return fallback
}

func intFromEnv(getenv func(string) string, key string, fallback int) int {
	value := strings.TrimSpace(getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
