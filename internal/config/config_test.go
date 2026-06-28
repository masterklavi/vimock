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
}

func TestLoadEnvAndFlagOverride(t *testing.T) {
	env := map[string]string{
		"VIMOCK_HOST": "127.0.0.1",
		"VIMOCK_PORT": "9090",
	}

	cfg, err := load([]string{"--port", "9191"}, func(key string) string {
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

func TestAddr(t *testing.T) {
	cfg := Config{Host: "127.0.0.1", Port: 8080}
	if got := cfg.Addr(); got != "127.0.0.1:8080" {
		t.Fatalf("addr = %q, want 127.0.0.1:8080", got)
	}
}
