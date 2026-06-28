package config

import "testing"

func TestLoadPublicAndBoolEnvFallbacks(t *testing.T) {
	t.Setenv("VIMOCK_PORT", "not-int")
	cfg, err := Load([]string{"--port", "8082"})
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if cfg.Port != 8082 {
		t.Fatalf("port = %d", cfg.Port)
	}
	if boolFromEnv(func(string) string { return "maybe" }, "X", true) != true {
		t.Fatal("invalid bool should return fallback true")
	}
	if boolFromEnv(func(string) string { return "off" }, "X", true) != false {
		t.Fatal("off should parse false")
	}
}
