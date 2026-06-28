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
	Host          string
	Port          int
	HTTPSPort     int
	TLSCertFile   string
	TLSKeyFile    string
	TLSSelfSigned bool
	Version       bool
}

func (c Config) Addr() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

func (c Config) HTTPSAddr() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.HTTPSPort))
}

func (c Config) HTTPSEnabled() bool {
	return c.HTTPSPort > 0
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
		Host:          stringFromEnv(getenv, "VIMOCK_HOST", defaultHost),
		Port:          intFromEnv(getenv, "VIMOCK_PORT", defaultPort),
		HTTPSPort:     intFromEnv(getenv, "VIMOCK_HTTPS_PORT", 0),
		TLSCertFile:   stringFromEnv(getenv, "VIMOCK_TLS_CERT_FILE", ""),
		TLSKeyFile:    stringFromEnv(getenv, "VIMOCK_TLS_KEY_FILE", ""),
		TLSSelfSigned: boolFromEnv(getenv, "VIMOCK_TLS_SELF_SIGNED", false),
	}

	fs := flag.NewFlagSet("vimock", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.StringVar(&cfg.Host, "host", cfg.Host, "bind host, overrides VIMOCK_HOST")
	fs.IntVar(&cfg.Port, "port", cfg.Port, "bind port, overrides VIMOCK_PORT")
	fs.IntVar(&cfg.HTTPSPort, "https-port", cfg.HTTPSPort, "HTTPS bind port, disabled by default, overrides VIMOCK_HTTPS_PORT")
	fs.StringVar(&cfg.TLSCertFile, "tls-cert-file", cfg.TLSCertFile, "TLS certificate file for HTTPS, overrides VIMOCK_TLS_CERT_FILE")
	fs.StringVar(&cfg.TLSKeyFile, "tls-key-file", cfg.TLSKeyFile, "TLS private key file for HTTPS, overrides VIMOCK_TLS_KEY_FILE")
	fs.BoolVar(&cfg.TLSSelfSigned, "tls-self-signed", cfg.TLSSelfSigned, "generate an in-memory self-signed certificate for HTTPS, overrides VIMOCK_TLS_SELF_SIGNED")
	fs.BoolVar(&cfg.Version, "version", false, "print version and exit")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if cfg.Host = strings.TrimSpace(cfg.Host); cfg.Host == "" {
		return Config{}, fmt.Errorf("host must not be empty")
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return Config{}, fmt.Errorf("port must be between 1 and 65535")
	}
	cfg.TLSCertFile = strings.TrimSpace(cfg.TLSCertFile)
	cfg.TLSKeyFile = strings.TrimSpace(cfg.TLSKeyFile)
	if cfg.HTTPSPort < 0 || cfg.HTTPSPort > 65535 {
		return Config{}, fmt.Errorf("https-port must be between 1 and 65535 when set")
	}
	if cfg.HTTPSPort == 0 {
		if cfg.TLSCertFile != "" || cfg.TLSKeyFile != "" || cfg.TLSSelfSigned {
			return Config{}, fmt.Errorf("https-port must be set when TLS options are used")
		}
		return cfg, nil
	}
	if cfg.TLSCertFile != "" || cfg.TLSKeyFile != "" {
		if cfg.TLSCertFile == "" || cfg.TLSKeyFile == "" {
			return Config{}, fmt.Errorf("tls-cert-file and tls-key-file must be set together")
		}
		return cfg, nil
	}
	if !cfg.TLSSelfSigned {
		return Config{}, fmt.Errorf("https requires tls-cert-file/tls-key-file or tls-self-signed")
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

func boolFromEnv(getenv func(string) string, key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(getenv(key)))
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "t", "yes", "y", "on":
		return true
	case "0", "false", "f", "no", "n", "off":
		return false
	default:
		return fallback
	}
}
