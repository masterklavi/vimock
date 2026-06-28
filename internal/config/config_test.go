package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	cfg, err := load(nil, func(string) string { return "" }, nil)
	if err != nil {
		t.Fatalf("load defaults: %v", err)
	}

	if cfg.Host != defaultHost {
		t.Fatalf("host = %q, want %q", cfg.Host, defaultHost)
	}
	if cfg.Port != defaultPort {
		t.Fatalf("port = %d, want %d", cfg.Port, defaultPort)
	}
	if cfg.HTTPSEnabled() {
		t.Fatal("HTTPS should be disabled by default")
	}
}

func TestLoadEnvAndFlagOverride(t *testing.T) {
	env := map[string]string{
		"VIMOCK_HOST":            "127.0.0.1",
		"VIMOCK_PORT":            "9090",
		"VIMOCK_HTTPS_PORT":      "9443",
		"VIMOCK_TLS_SELF_SIGNED": "true",
	}

	cfg, err := load([]string{"--port", "9191", "--https-port", "8443"}, func(key string) string {
		return env[key]
	}, nil)
	if err != nil {
		t.Fatalf("load env and flags: %v", err)
	}

	if cfg.Host != "127.0.0.1" {
		t.Fatalf("host = %q, want env value", cfg.Host)
	}
	if cfg.Port != 9191 {
		t.Fatalf("port = %d, want flag override", cfg.Port)
	}
	if cfg.HTTPSPort != 8443 {
		t.Fatalf("https port = %d, want flag override", cfg.HTTPSPort)
	}
	if !cfg.TLSSelfSigned {
		t.Fatal("self-signed TLS flag should be loaded from env")
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	_, err := load([]string{"--port", "70000"}, func(string) string { return "" }, nil)
	if err == nil {
		t.Fatal("expected invalid port error")
	}
}

func TestLoadVersionFlag(t *testing.T) {
	cfg, err := load([]string{"--version"}, func(string) string { return "" }, nil)
	if err != nil {
		t.Fatalf("load version flag: %v", err)
	}
	if !cfg.Version {
		t.Fatalf("version = false, want true")
	}
}

func TestLoadHTTPSWithCertFiles(t *testing.T) {
	cfg, err := load([]string{
		"--https-port", "8443",
		"--tls-cert-file", "cert.pem",
		"--tls-key-file", "key.pem",
	}, func(string) string { return "" }, nil)
	if err != nil {
		t.Fatalf("load https cert files: %v", err)
	}
	if !cfg.HTTPSEnabled() {
		t.Fatal("HTTPS should be enabled")
	}
	if cfg.HTTPSAddr() != "0.0.0.0:8443" {
		t.Fatalf("https addr = %q, want 0.0.0.0:8443", cfg.HTTPSAddr())
	}
}

func TestLoadRejectsHTTPSWithoutTLSMaterial(t *testing.T) {
	_, err := load([]string{"--https-port", "8443"}, func(string) string { return "" }, nil)
	if err == nil {
		t.Fatal("expected HTTPS without TLS material error")
	}
}

func TestLoadRejectsPartialCertFiles(t *testing.T) {
	_, err := load([]string{"--https-port", "8443", "--tls-cert-file", "cert.pem"}, func(string) string { return "" }, nil)
	if err == nil {
		t.Fatal("expected partial TLS file error")
	}
}

func TestLoadRejectsTLSOptionsWithoutHTTPSPort(t *testing.T) {
	_, err := load([]string{"--tls-self-signed"}, func(string) string { return "" }, nil)
	if err == nil {
		t.Fatal("expected TLS option without HTTPS port error")
	}
}

func TestAddr(t *testing.T) {
	cfg := Config{Host: "127.0.0.1", Port: 8080}
	if got := cfg.Addr(); got != "127.0.0.1:8080" {
		t.Fatalf("addr = %q, want 127.0.0.1:8080", got)
	}
}
