package tlsconfig

import (
	"crypto/tls"
	"testing"
	"time"
)

func TestLoadSelfSignedConfig(t *testing.T) {
	cfg, err := Load("", "", true, []string{"localhost", "127.0.0.1"})
	if err != nil {
		t.Fatalf("load self-signed config: %v", err)
	}
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Fatalf("min version = %x, want TLS 1.2", cfg.MinVersion)
	}
	if len(cfg.Certificates) != 1 {
		t.Fatalf("certificates = %d, want 1", len(cfg.Certificates))
	}
	if len(cfg.NextProtos) != 2 || cfg.NextProtos[0] != "h2" || cfg.NextProtos[1] != "http/1.1" {
		t.Fatalf("next protos = %#v", cfg.NextProtos)
	}
}

func TestNewSelfSignedCertificateIncludesHosts(t *testing.T) {
	cert, err := NewSelfSignedCertificate([]string{"localhost", "127.0.0.1"}, time.Unix(1000, 0))
	if err != nil {
		t.Fatalf("new self-signed cert: %v", err)
	}
	if cert.Leaf == nil {
		t.Fatal("leaf certificate should be parsed")
	}
	if got := cert.Leaf.VerifyHostname("localhost"); got != nil {
		t.Fatalf("verify localhost: %v", got)
	}
	if got := cert.Leaf.VerifyHostname("127.0.0.1"); got != nil {
		t.Fatalf("verify 127.0.0.1: %v", got)
	}
}

func TestLoadRejectsMissingMaterial(t *testing.T) {
	_, err := Load("", "", false, nil)
	if err == nil {
		t.Fatal("expected missing TLS material error")
	}
}

func TestLoadRejectsPartialFiles(t *testing.T) {
	_, err := Load("cert.pem", "", false, nil)
	if err == nil {
		t.Fatal("expected partial TLS files error")
	}
}
