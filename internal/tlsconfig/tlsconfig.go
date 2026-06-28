package tlsconfig

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"
)

// Load returns a server TLS config for VIMock HTTPS listeners.
func Load(certFile, keyFile string, selfSigned bool, hosts []string) (*tls.Config, error) {
	var cert tls.Certificate
	var err error

	switch {
	case certFile != "" || keyFile != "":
		if certFile == "" || keyFile == "" {
			return nil, fmt.Errorf("tls cert and key files must be set together")
		}
		cert, err = tls.LoadX509KeyPair(certFile, keyFile)
	case selfSigned:
		cert, err = NewSelfSignedCertificate(hosts, time.Now())
	default:
		return nil, fmt.Errorf("tls cert/key files or self-signed mode must be configured")
	}
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		NextProtos:   []string{"h2", "http/1.1"},
		Certificates: []tls.Certificate{cert},
	}, nil
}

// NewSelfSignedCertificate generates an in-memory certificate for local/CI HTTPS smoke checks.
func NewSelfSignedCertificate(hosts []string, now time.Time) (tls.Certificate, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate private key: %w", err)
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate serial number: %w", err)
	}

	certificate := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "vimock.local",
		},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	addHosts(&certificate, hosts)
	if len(certificate.DNSNames) == 0 && len(certificate.IPAddresses) == 0 {
		certificate.DNSNames = []string{"localhost"}
		certificate.IPAddresses = []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}
	}

	der, err := x509.CreateCertificate(rand.Reader, &certificate, &certificate, privateKey.Public(), privateKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("marshal private key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("load generated key pair: %w", err)
	}
	cert.Leaf, _ = x509.ParseCertificate(der)
	return cert, nil
}

func addHosts(certificate *x509.Certificate, hosts []string) {
	seenDNS := make(map[string]struct{})
	seenIP := make(map[string]struct{})
	for _, host := range hosts {
		host = strings.TrimSpace(host)
		if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
			continue
		}
		if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
			host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
		}
		if ip := net.ParseIP(host); ip != nil {
			key := ip.String()
			if _, ok := seenIP[key]; ok {
				continue
			}
			certificate.IPAddresses = append(certificate.IPAddresses, ip)
			seenIP[key] = struct{}{}
			continue
		}
		if _, ok := seenDNS[host]; ok {
			continue
		}
		certificate.DNSNames = append(certificate.DNSNames, host)
		seenDNS[host] = struct{}{}
	}
}
