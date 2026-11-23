// Package testutil provides testing utilities and helpers for the redis-tui application.
package testutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/davidbudnick/redis-tui/internal/db"
	"github.com/davidbudnick/redis-tui/internal/types"
)

// Injectable for tests to simulate crypto failures.
var (
	ecdsaGenerateKey      = ecdsa.GenerateKey
	x509CreateCertificate = x509.CreateCertificate
)

// Indirections so tests can cover the failure branches of helpers that would
// otherwise depend on infallible operations.
var (
	dbNewConfig         = db.NewConfig
	listConnectionsFunc = func(cfg *db.Config) ([]types.Connection, error) {
		return cfg.ListConnections()
	}
)

// TempConfigPath creates a temporary config file path for testing.
// The file and directory will be cleaned up after the test.
func TempConfigPath(t testing.TB) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "config.json")
}

// NewTestConfig creates a new Config instance using a temporary file for testing.
func NewTestConfig(t testing.TB) *db.Config {
	t.Helper()
	path := TempConfigPath(t)
	cfg, err := dbNewConfig(path)
	if err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}
	return cfg
}

// MustAddConnection adds a connection to the config or fails the test.
func MustAddConnection(t testing.TB, cfg *db.Config, conn types.Connection) types.Connection {
	t.Helper()
	conn, err := cfg.AddConnection(conn)
	if err != nil {
		t.Fatalf("failed to add connection: %v", err)
	}
	return conn
}

// AssertConnectionExists checks that a connection with the given ID exists.
func AssertConnectionExists(t testing.TB, cfg *db.Config, id int64) types.Connection {
	t.Helper()
	connections, err := listConnectionsFunc(cfg)
	if err != nil {
		t.Fatalf("failed to list connections: %v", err)
	}
	for _, conn := range connections {
		if conn.ID == id {
			return conn
		}
	}
	t.Fatalf("connection with ID %d not found", id)
	return types.Connection{}
}

// AssertConnectionNotExists checks that a connection with the given ID does not exist.
func AssertConnectionNotExists(t testing.TB, cfg *db.Config, id int64) {
	t.Helper()
	connections, err := listConnectionsFunc(cfg)
	if err != nil {
		t.Fatalf("failed to list connections: %v", err)
	}
	for _, conn := range connections {
		if conn.ID == id {
			t.Fatalf("connection with ID %d should not exist", id)
		}
	}
}

// AssertEqual checks if two values are equal.
func AssertEqual[T comparable](t testing.TB, got, want T, msg string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %v, want %v", msg, got, want)
	}
}

// AssertNoError checks that an error is nil.
func AssertNoError(t testing.TB, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Errorf("%s: unexpected error: %v", msg, err)
	}
}

// AssertError checks that an error is not nil.
func AssertError(t testing.TB, err error, msg string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: expected error but got nil", msg)
	}
}

// AssertSliceLen checks that a slice has the expected length.
func AssertSliceLen[T any](t testing.TB, slice []T, expectedLen int, msg string) {
	t.Helper()
	if len(slice) != expectedLen {
		t.Errorf("%s: got slice length %d, want %d", msg, len(slice), expectedLen)
	}
}

// FileExists checks if a file exists at the given path.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// SampleConnection returns a sample connection for testing.
func SampleConnection() types.Connection {
	return types.Connection{
		ID:       1,
		Name:     "test-redis",
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
	}
}

// SampleRedisKey returns a sample Redis key for testing.
func SampleRedisKey(name string, keyType types.KeyType) types.RedisKey {
	return types.RedisKey{
		Key:  name,
		Type: keyType,
		TTL:  -1, // No expiration
	}
}

// SampleFavorite returns a sample favorite for testing.
func SampleFavorite(connID int64, key string) types.Favorite {
	return types.Favorite{
		ConnectionID: connID,
		Key:          key,
		Label:        "Test Favorite",
	}
}

func generateEphemeralCertOrError(r io.Reader) (tls.Certificate, error) {
	priv, err := ecdsaGenerateKey(elliptic.P256(), r)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate key: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Redis TUI Test"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(1 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509CreateCertificate(r, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create certificate: %w", err)
	}

	return tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  priv,
	}, nil
}

// GenerateEphemeralCert generates an ephemeral TLS certificate for testing.
func GenerateEphemeralCert(t testing.TB) tls.Certificate {
	t.Helper()
	cert, err := generateEphemeralCertOrError(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate ephemeral TLS certificate: %v", err)
	}
	return cert
}
