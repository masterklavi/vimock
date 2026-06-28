package tlsconfig

import (
	"os"
	"testing"
	"time"
)

func TestSelfSignedFallbackHostsAndSkippedBindHosts(t *testing.T) {
	cert, err := NewSelfSignedCertificate([]string{"0.0.0.0", "::", "[::]", "example.local", "example.local"}, time.Unix(1000, 0))
	if err != nil {
		t.Fatalf("NewSelfSignedCertificate(): %v", err)
	}
	if err := cert.Leaf.VerifyHostname("example.local"); err != nil {
		t.Fatalf("verify example.local: %v", err)
	}
	cert, err = NewSelfSignedCertificate([]string{"0.0.0.0"}, time.Unix(1000, 0))
	if err != nil {
		t.Fatalf("fallback cert: %v", err)
	}
	if err := cert.Leaf.VerifyHostname("localhost"); err != nil {
		t.Fatalf("fallback localhost: %v", err)
	}
}

func TestSelfSignedAcceptsIPv6AndDeduplicatesHosts(t *testing.T) {
	cert, err := NewSelfSignedCertificate([]string{"[::1]", "::1", "127.0.0.1", "127.0.0.1", "api.local"}, time.Unix(1000, 0))
	if err != nil {
		t.Fatalf("NewSelfSignedCertificate(): %v", err)
	}
	for _, host := range []string{"::1", "127.0.0.1", "api.local"} {
		if err := cert.Leaf.VerifyHostname(host); err != nil {
			t.Fatalf("verify %s: %v", host, err)
		}
	}
	if len(cert.Leaf.IPAddresses) != 2 {
		t.Fatalf("IPAddresses = %v, want 2 unique addresses", cert.Leaf.IPAddresses)
	}
	if len(cert.Leaf.DNSNames) != 1 {
		t.Fatalf("DNSNames = %v, want 1 unique name", cert.Leaf.DNSNames)
	}
}

func TestLoadFileErrors(t *testing.T) {
	_, err := Load("missing.crt", "missing.key", false, nil)
	if err == nil {
		t.Fatal("missing cert files error = nil")
	}
	certFile, err := os.CreateTemp(t.TempDir(), "cert-*.pem")
	if err != nil {
		t.Fatal(err)
	}
	keyFile, err := os.CreateTemp(t.TempDir(), "key-*.pem")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = certFile.WriteString("bad")
	_, _ = keyFile.WriteString("bad")
	_ = certFile.Close()
	_ = keyFile.Close()
	_, err = Load(certFile.Name(), keyFile.Name(), false, nil)
	if err == nil {
		t.Fatal("bad cert files error = nil")
	}
}
