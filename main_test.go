package main

import (
	"flag"
	"testing"
)

func TestParseFlags_NoArgs(t *testing.T) {
	conn, version, err := parseFlags([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn != nil {
		t.Error("expected nil connection with no args")
	}
	if version {
		t.Error("expected version=false")
	}
}

func TestParseFlags_HostOnly(t *testing.T) {
	conn, _, err := parseFlags([]string{"--host", "localhost"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Host != "localhost" {
		t.Errorf("Host = %q, want %q", conn.Host, "localhost")
	}
	if conn.Port != 6379 {
		t.Errorf("Port = %d, want %d", conn.Port, 6379)
	}
	if conn.DB != 0 {
		t.Errorf("DB = %d, want %d", conn.DB, 0)
	}
	if conn.Name != "localhost:6379" {
		t.Errorf("Name = %q, want %q", conn.Name, "localhost:6379")
	}
}

func TestParseFlags_ShortFlags(t *testing.T) {
	conn, _, err := parseFlags([]string{"-h", "redis.example.com", "-p", "6380", "-a", "secret", "-n", "5"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Host != "redis.example.com" {
		t.Errorf("Host = %q, want %q", conn.Host, "redis.example.com")
	}
	if conn.Port != 6380 {
		t.Errorf("Port = %d, want %d", conn.Port, 6380)
	}
	if conn.Password != "secret" {
		t.Errorf("Password = %q, want %q", conn.Password, "secret")
	}
	if conn.DB != 5 {
		t.Errorf("DB = %d, want %d", conn.DB, 5)
	}
}

func TestParseFlags_LongFlags(t *testing.T) {
	conn, _, err := parseFlags([]string{"--host", "10.0.0.1", "--port", "7000", "--password", "pass", "--db", "3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Host != "10.0.0.1" {
		t.Errorf("Host = %q, want %q", conn.Host, "10.0.0.1")
	}
	if conn.Port != 7000 {
		t.Errorf("Port = %d, want %d", conn.Port, 7000)
	}
	if conn.Password != "pass" {
		t.Errorf("Password = %q, want %q", conn.Password, "pass")
	}
	if conn.DB != 3 {
		t.Errorf("DB = %d, want %d", conn.DB, 3)
	}
}

func TestParseFlags_CustomName(t *testing.T) {
	conn, _, err := parseFlags([]string{"--host", "localhost", "--name", "Production"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Name != "Production" {
		t.Errorf("Name = %q, want %q", conn.Name, "Production")
	}
}

func TestParseFlags_DefaultName(t *testing.T) {
	conn, _, err := parseFlags([]string{"--host", "myhost", "--port", "9999"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Name != "myhost:9999" {
		t.Errorf("Name = %q, want %q", conn.Name, "myhost:9999")
	}
}

func TestParseFlags_Cluster(t *testing.T) {
	conn, _, err := parseFlags([]string{"--host", "localhost", "--cluster"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if !conn.UseCluster {
		t.Error("UseCluster should be true")
	}
}

func TestParseFlags_TLS(t *testing.T) {
	conn, _, err := parseFlags([]string{
		"--host", "localhost",
		"--tls",
		"--tls-cert", "/path/cert.pem",
		"--tls-key", "/path/key.pem",
		"--tls-ca", "/path/ca.pem",
		"--tls-skip-verify",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if !conn.UseTLS {
		t.Error("UseTLS should be true")
	}
	if conn.TLSConfig == nil {
		t.Fatal("TLSConfig should be set")
	}
	if conn.TLSConfig.CertFile != "/path/cert.pem" {
		t.Errorf("CertFile = %q, want %q", conn.TLSConfig.CertFile, "/path/cert.pem")
	}
	if conn.TLSConfig.KeyFile != "/path/key.pem" {
		t.Errorf("KeyFile = %q, want %q", conn.TLSConfig.KeyFile, "/path/key.pem")
	}
	if conn.TLSConfig.CAFile != "/path/ca.pem" {
		t.Errorf("CAFile = %q, want %q", conn.TLSConfig.CAFile, "/path/ca.pem")
	}
	if !conn.TLSConfig.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should be true")
	}
}

func TestParseFlags_TLSNotSet(t *testing.T) {
	conn, _, err := parseFlags([]string{"--host", "localhost"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.UseTLS {
		t.Error("UseTLS should be false")
	}
	if conn.TLSConfig != nil {
		t.Error("TLSConfig should be nil when --tls is not set")
	}
}

func TestParseFlags_Version(t *testing.T) {
	conn, version, err := parseFlags([]string{"--version"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn != nil {
		t.Error("expected nil connection for --version")
	}
	if !version {
		t.Error("expected version=true")
	}
}

func TestParseFlags_AllOptions(t *testing.T) {
	conn, _, err := parseFlags([]string{
		"--host", "redis.prod.com",
		"--port", "6380",
		"--password", "s3cret",
		"--db", "7",
		"--name", "Prod Redis",
		"--cluster",
		"--tls",
		"--tls-cert", "/cert.pem",
		"--tls-key", "/key.pem",
		"--tls-ca", "/ca.pem",
		"--tls-skip-verify",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if conn.Host != "redis.prod.com" {
		t.Errorf("Host = %q", conn.Host)
	}
	if conn.Port != 6380 {
		t.Errorf("Port = %d", conn.Port)
	}
	if conn.Password != "s3cret" {
		t.Errorf("Password = %q", conn.Password)
	}
	if conn.DB != 7 {
		t.Errorf("DB = %d", conn.DB)
	}
	if conn.Name != "Prod Redis" {
		t.Errorf("Name = %q", conn.Name)
	}
	if !conn.UseCluster {
		t.Error("UseCluster should be true")
	}
	if !conn.UseTLS {
		t.Error("UseTLS should be true")
	}
	if conn.TLSConfig == nil {
		t.Fatal("TLSConfig should be set")
	}
}

func TestParseFlags_InvalidFlag(t *testing.T) {
	_, _, err := parseFlags([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestParseFlags_Help(t *testing.T) {
	_, _, err := parseFlags([]string{"--help"})
	if err != flag.ErrHelp {
		t.Errorf("expected flag.ErrHelp, got %v", err)
	}
}
