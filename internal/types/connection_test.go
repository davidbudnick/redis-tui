package types

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildTLSConfig_Minimal(t *testing.T) {
	tc := &TLSConfig{}
	cfg, err := tc.BuildTLSConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should be false by default")
	}
	if cfg.ServerName != "" {
		t.Errorf("ServerName = %q, want empty", cfg.ServerName)
	}
}

func TestBuildTLSConfig_InsecureSkipVerify(t *testing.T) {
	tc := &TLSConfig{InsecureSkipVerify: true}
	cfg, err := tc.BuildTLSConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should be true")
	}
}

func TestBuildTLSConfig_ServerName(t *testing.T) {
	tc := &TLSConfig{ServerName: "redis.example.com"}
	cfg, err := tc.BuildTLSConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ServerName != "redis.example.com" {
		t.Errorf("ServerName = %q, want %q", cfg.ServerName, "redis.example.com")
	}
}

func TestBuildTLSConfig_BadCertFile(t *testing.T) {
	tc := &TLSConfig{
		CertFile: "/nonexistent/cert.pem",
		KeyFile:  "/nonexistent/key.pem",
	}
	_, err := tc.BuildTLSConfig()
	if err == nil {
		t.Error("expected error for nonexistent cert file")
	}
}

func TestBuildTLSConfig_BadCAFile(t *testing.T) {
	tc := &TLSConfig{
		CAFile: "/nonexistent/ca.pem",
	}
	_, err := tc.BuildTLSConfig()
	if err == nil {
		t.Error("expected error for nonexistent CA file")
	}
}

func TestBuildTLSConfig_InvalidCAPEM(t *testing.T) {
	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(caPath, []byte("not a valid PEM"), 0600); err != nil {
		t.Fatal(err)
	}

	tc := &TLSConfig{CAFile: caPath}
	_, err := tc.BuildTLSConfig()
	if err == nil {
		t.Error("expected error for invalid CA PEM data")
	}
}

func TestBuildTLSConfig_ValidCertAndCA(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath, caPath := generateTestCerts(t, dir)

	tc := &TLSConfig{
		CertFile: certPath,
		KeyFile:  keyPath,
		CAFile:   caPath,
	}
	cfg, err := tc.BuildTLSConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Certificates) != 1 {
		t.Errorf("expected 1 certificate, got %d", len(cfg.Certificates))
	}
	if cfg.RootCAs == nil {
		t.Error("RootCAs should be set")
	}
}

func TestBuildTLSConfig_CertWithoutKey(t *testing.T) {
	dir := t.TempDir()
	certPath, _, _ := generateTestCerts(t, dir)

	// Only cert, no key — should still work (CertFile is set but KeyFile is empty,
	// so the cert/key pair loading is skipped)
	tc := &TLSConfig{CertFile: certPath}
	cfg, err := tc.BuildTLSConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Certificates) != 0 {
		t.Error("certificates should be empty when only CertFile is set without KeyFile")
	}
}

// generateTestCerts creates a self-signed CA and client cert/key for testing.
func generateTestCerts(t *testing.T, dir string) (certPath, keyPath, caPath string) {
	t.Helper()

	// Generate CA key
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}

	caPath = filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}), 0600); err != nil {
		t.Fatal(err)
	}

	// Generate client key
	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "Test Client"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
	}
	clientDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caTemplate, &clientKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}

	certPath = filepath.Join(dir, "cert.pem")
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientDER}), 0600); err != nil {
		t.Fatal(err)
	}

	keyDER, err := x509.MarshalECPrivateKey(clientKey)
	if err != nil {
		t.Fatal(err)
	}
	keyPath = filepath.Join(dir, "key.pem")
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0600); err != nil {
		t.Fatal(err)
	}

	return certPath, keyPath, caPath
}
