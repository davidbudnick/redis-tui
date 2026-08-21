package redis

import (
	"context"
	"crypto/tls"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/davidbudnick/redis-tui/internal/testutil"
	"github.com/davidbudnick/redis-tui/internal/types"
)

type stubPasswordResolver struct {
	values    map[string]string
	err       error
	path      string
	selectors []string
}

func (r *stubPasswordResolver) Resolve(_ context.Context, path string, selectors ...string) (map[string]string, error) {
	r.path = path
	r.selectors = selectors
	return r.values, r.err
}

func TestResolveCredentials(t *testing.T) {
	client := NewClient()

	plain := types.Connection{Password: "literal"}
	got, err := client.resolveCredentials(plain)
	if err != nil || got.Password != "literal" {
		t.Fatalf("plain password resolution = %#v, %v", got, err)
	}

	if _, err := client.resolveCredentials(types.Connection{VaultPasswordKey: "password"}); err == nil {
		t.Fatal("expected missing Vault path error")
	}
	if _, err := client.resolveCredentials(types.Connection{VaultPath: "secret/data/redis"}); err == nil {
		t.Fatal("expected missing Vault key error")
	}

	userKey := "credentials.VALKEY.username"
	passwordKey := "credentials.VALKEY.password"
	resolver := &stubPasswordResolver{values: map[string]string{userKey: "from-vault-user", passwordKey: "from-vault"}}
	client.credentials = resolver
	got, err = client.resolveCredentials(types.Connection{Username: "ignored", Password: "ignored", VaultPath: "secret/data/redis", VaultUserKey: userKey, VaultPasswordKey: passwordKey})
	if err != nil {
		t.Fatalf("resolveCredentials() error = %v", err)
	}
	if got.Username != "from-vault-user" {
		t.Errorf("Username = %q, want %q", got.Username, "from-vault-user")
	}
	if got.Password != "from-vault" {
		t.Errorf("Password = %q, want %q", got.Password, "from-vault")
	}
	if resolver.path != "secret/data/redis" || len(resolver.selectors) != 2 || resolver.selectors[0] != userKey || resolver.selectors[1] != passwordKey {
		t.Errorf("resolver called with path %q and selectors %v", resolver.path, resolver.selectors)
	}

	client.credentials = &stubPasswordResolver{err: errors.New("Vault unavailable")}
	if _, err := client.resolveCredentials(types.Connection{VaultPath: "secret/data/redis", VaultPasswordKey: "password"}); err == nil || !strings.Contains(err.Error(), "resolve Redis credentials from Vault") {
		t.Fatalf("resolveCredentials() error = %v", err)
	}

	resolver = &stubPasswordResolver{values: map[string]string{userKey: "user-only"}}
	client.credentials = resolver
	got, err = client.resolveCredentials(types.Connection{Password: "literal", VaultPath: "secret/data/redis", VaultUserKey: userKey})
	if err != nil || got.Username != "user-only" || got.Password != "literal" {
		t.Fatalf("username-only resolution = %#v, %v", got, err)
	}
}

func TestVaultPasswordConnectionFlows(t *testing.T) {
	resolverErr := errors.New("Vault unavailable")
	conn := types.Connection{Host: "localhost", Port: 6379, VaultPath: "secret/data/redis", VaultPasswordKey: "password"}

	client := NewClient()
	client.credentials = &stubPasswordResolver{err: resolverErr}
	if err := client.Connect(conn); !errors.Is(err, resolverErr) {
		t.Errorf("Connect() error = %v, want resolver error", err)
	}
	if err := client.ConnectCluster([]string{"localhost:6379"}, conn); !errors.Is(err, resolverErr) {
		t.Errorf("ConnectCluster() error = %v, want resolver error", err)
	}
	if _, err := client.TestConnection(conn); !errors.Is(err, resolverErr) {
		t.Errorf("TestConnection() error = %v, want resolver error", err)
	}

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	mr.RequireAuth("from-vault")
	port, err := strconv.Atoi(mr.Port())
	if err != nil {
		t.Fatalf("parse miniredis port: %v", err)
	}

	client = NewClient()
	client.credentials = &stubPasswordResolver{values: map[string]string{"password": "from-vault"}}
	conn.Host = mr.Host()
	conn.Port = port
	if err := client.Connect(conn); err != nil {
		t.Fatalf("Connect() with Vault password error = %v", err)
	}
	t.Cleanup(func() {
		if err := client.Disconnect(); err != nil {
			t.Errorf("Disconnect() error = %v", err)
		}
	})
}

func TestParseAddr(t *testing.T) {
	tests := []struct {
		name         string
		addr         string
		expectedHost string
		expectedPort int
	}{
		{"host:port", "localhost:6379", "localhost", 6379},
		{"custom port", "myhost:6380", "myhost", 6380},
		{"hostname only no port", "hostname", "hostname", 6379},
		{"empty string", "", "", 6379},
		{"ip with port", "192.168.1.1:7000", "192.168.1.1", 7000},
		{"ip without port", "192.168.1.1", "192.168.1.1", 6379},
		{"port 0", "host:0", "host", 0},
		{"non-numeric port uses default", "host:abc", "host", 6379},
		{"ipv6 with port", "[::1]:6379", "::1", 6379},
		{"ipv6 with custom port", "[2001:db8::1]:7000", "2001:db8::1", 7000},
		{"ipv6 without port", "::1", "::1", 6379},
		{"ipv6 bracketed without port", "[::1]", "[::1]", 6379},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port := parseAddr(tt.addr)
			if host != tt.expectedHost {
				t.Errorf("parseAddr(%q) host = %q, want %q", tt.addr, host, tt.expectedHost)
			}
			if port != tt.expectedPort {
				t.Errorf("parseAddr(%q) port = %d, want %d", tt.addr, port, tt.expectedPort)
			}
		})
	}
}

func TestConnect(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := NewClient()
	port, _ := strconv.Atoi(mr.Port())

	if err := client.Connect(types.Connection{Name: "test", Host: mr.Host(), Port: port, Username: "default", Password: "", DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect() returned error: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect() })

	// Verify the client is usable by setting and getting a key
	mr.Set("testkey", "testval")
	got, err := client.client.Get(client.ctx, "testkey").Result()
	if err != nil {
		t.Fatalf("Get after Connect failed: %v", err)
	}
	if got != "testval" {
		t.Errorf("Get = %q, want %q", got, "testval")
	}
}

func TestConnect_WrongPassword(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	mr.RequireAuth("correct-password")

	client := NewClient()
	port, _ := strconv.Atoi(mr.Port())

	err = client.Connect(types.Connection{Name: "test", Host: mr.Host(), Port: port, Username: "default", Password: "wrong-password", DB: 0, UseCluster: false})
	if err == nil {
		_ = client.Disconnect()
		t.Fatal("expected error when connecting with wrong password")
	}
}

func TestConnect_WrongUsernamePassword(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	mr.RequireUserAuth("testuser", "correct-password")

	client := NewClient()
	port, _ := strconv.Atoi(mr.Port())

	err = client.Connect(types.Connection{Name: "test", Host: mr.Host(), Port: port, Username: "wrong-user", Password: "wrong-password", DB: 0, UseCluster: false})
	if err == nil {
		_ = client.Disconnect()
		t.Fatal("expected error when connecting with wrong username and password")
	}
}

func TestConnect_EmptyHostError(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := NewClient()
	port, _ := strconv.Atoi(mr.Port())

	err = client.Connect(types.Connection{Name: "test", Host: "", Port: port, Password: "wrong-password", DB: 0, UseCluster: false})
	if err == nil {
		_ = client.Disconnect()
		t.Fatal("expected error when connecting with empty host")
	}
}

func TestConnect_EmptyPortError(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := NewClient()
	port := 0

	err = client.Connect(types.Connection{Name: "", Host: mr.Host(), Port: port, Password: "wrong-password", DB: 0, UseCluster: false})
	if err == nil {
		_ = client.Disconnect()
		t.Fatal("expected error when connecting with empty port")
	}
}

func TestConnect_CorrectPassword(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	mr.RequireAuth("correct-password")

	client := NewClient()
	port, _ := strconv.Atoi(mr.Port())

	err = client.Connect(types.Connection{Name: "test", Host: mr.Host(), Port: port, Username: "default", Password: "correct-password", DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("Connect() with correct password returned error: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect() })
}

func TestConnect_CorrectUserPassword(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	mr.RequireUserAuth("testuser", "correct-password")

	client := NewClient()
	port, _ := strconv.Atoi(mr.Port())

	err = client.Connect(types.Connection{Name: "test", Host: mr.Host(), Port: port, Username: "testuser", Password: "correct-password", DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("Connect() with correct username/password returned error: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect() })
}

func TestConnect_WithTLS(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		serverCert := testutil.GenerateEphemeralCert(t)

		serverTLSConfig := &tls.Config{
			Certificates: []tls.Certificate{serverCert},
		}

		mr, err := miniredis.RunTLS(serverTLSConfig)
		if err != nil {
			t.Fatalf("failed to start miniredis with TLS: %v", err)
		}
		t.Cleanup(mr.Close)

		client := NewClient()
		port, _ := strconv.Atoi(mr.Port())

		conn := types.Connection{
			Name:   "tls-test",
			Host:   mr.Host(),
			Port:   port,
			UseTLS: true,
			TLSConfig: &types.TLSConfig{
				InsecureSkipVerify: true,
			},
		}
		if err := client.Connect(conn); err != nil {
			t.Fatalf("Connect() with TLS returned error: %v", err)
		}
		t.Cleanup(func() { _ = client.Disconnect() })

		// Verify the client is usable by setting and getting a key
		mr.Set("tlskey", "tlsval")
		got, err := client.client.Get(client.ctx, "tlskey").Result()
		if err != nil {
			t.Fatalf("Get after TLS Connect failed: %v", err)
		}
		if got != "tlsval" {
			t.Errorf("Get = %q, want %q", got, "tlsval")
		}
	})
	t.Run("TLS requested but config is missing", func(t *testing.T) {
		client := NewClient()

		conn := types.Connection{
			Name:   "tls-missing-config",
			Host:   "localhost",
			Port:   6379,
			UseTLS: true,
			// TLSConfig is intentionally left nil
		}

		err := client.Connect(conn)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		expectedErr := "TLS requested but TLS configuration is missing"
		if err.Error() != expectedErr {
			t.Errorf("expected error %q, got %q", expectedErr, err.Error())
		}
	})
	t.Run("failed to build TLS config", func(t *testing.T) {
		client := NewClient()

		conn := types.Connection{
			Name:   "tls-build-error",
			Host:   "localhost",
			Port:   6379,
			UseTLS: true,
			TLSConfig: &types.TLSConfig{
				CertFile: "/path/to/nowhere/cert.pem",
				KeyFile:  "/path/to/nowhere/key.pem",
			},
		}

		err := client.Connect(conn)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !strings.Contains(err.Error(), "failed to build TLS config") &&
			!strings.Contains(err.Error(), "failed to load TLS key pair") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestDisconnect(t *testing.T) {
	t.Run("disconnect after connect", func(t *testing.T) {
		mr, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to start miniredis: %v", err)
		}
		t.Cleanup(mr.Close)

		client := NewClient()
		port, _ := strconv.Atoi(mr.Port())

		if err := client.Connect(types.Connection{Name: "test", Host: mr.Host(), Port: port, Password: "", DB: 0, UseCluster: false}); err != nil {
			t.Fatalf("Connect() returned error: %v", err)
		}

		if err := client.Disconnect(); err != nil {
			t.Fatalf("Disconnect() returned error: %v", err)
		}

		if client.client != nil {
			t.Error("client.client should be nil after Disconnect")
		}
		if client.isCluster {
			t.Error("client.isCluster should be false after Disconnect")
		}
	})

	t.Run("disconnect on fresh client", func(t *testing.T) {
		client := NewClient()
		if err := client.Disconnect(); err != nil {
			t.Errorf("Disconnect() on fresh client returned error: %v", err)
		}
	})
}

func TestSelectDB(t *testing.T) {
	t.Run("select db succeeds", func(t *testing.T) {
		client, _ := setupTestClient(t)

		if err := client.SelectDB(1); err != nil {
			t.Fatalf("SelectDB(1) returned error: %v", err)
		}
		if client.db != 1 {
			t.Errorf("client.db = %d, want 1", client.db)
		}
	})

	t.Run("select db on cluster returns error", func(t *testing.T) {
		client, _ := setupTestClient(t)
		client.isCluster = true

		err := client.SelectDB(1)
		if err == nil {
			t.Fatal("SelectDB on cluster should return error")
		}
		if err.Error() != "database selection not supported in cluster mode" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("select db on nil client returns error", func(t *testing.T) {
		client := NewClient()

		err := client.SelectDB(1)
		if err == nil {
			t.Fatal("SelectDB on nil client should return error")
		}
		if err.Error() != "not connected" {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestCleanup(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := NewClient()
	port, _ := strconv.Atoi(mr.Port())

	if err := client.Connect(types.Connection{Name: "test", Host: mr.Host(), Port: port, Password: "", DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect() returned error: %v", err)
	}

	client.cleanup()

	if client.client != nil {
		t.Error("client.client should be nil after cleanup")
	}
}

func TestReconnectCycle(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := NewClient()
	port, _ := strconv.Atoi(mr.Port())

	// First connect
	if err := client.Connect(types.Connection{Name: "test", Host: mr.Host(), Port: port, Password: "", DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("first Connect() returned error: %v", err)
	}

	// Disconnect
	if err := client.Disconnect(); err != nil {
		t.Fatalf("Disconnect() returned error: %v", err)
	}

	// Reconnect
	if err := client.Connect(types.Connection{Name: "test", Host: mr.Host(), Port: port, Password: "", DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("second Connect() returned error: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect() })

	// Verify it works after reconnect
	mr.Set("reconnectkey", "reconnectval")
	got, err := client.client.Get(client.ctx, "reconnectkey").Result()
	if err != nil {
		t.Fatalf("Get after reconnect failed: %v", err)
	}
	if got != "reconnectval" {
		t.Errorf("Get = %q, want %q", got, "reconnectval")
	}
}

func TestConnectCluster_WithTLSSuccess(t *testing.T) {
	client := NewClient()
	t.Cleanup(func() { _ = client.Disconnect() })

	// Use a valid (empty) TLS config — BuildTLSConfig succeeds with no certs.
	// The cluster Ping will fail (no server), but opts.TLSConfig is set.
	conn := types.Connection{
		Name:      "cluster-tls-ok",
		Host:      "127.0.0.1",
		Port:      1,
		UseTLS:    true,
		TLSConfig: &types.TLSConfig{InsecureSkipVerify: true},
	}

	err := client.ConnectCluster([]string{"127.0.0.1:1"}, conn)
	// Expect a connection error (no server), not a TLS build error.
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
	if strings.Contains(err.Error(), "TLS") {
		t.Errorf("unexpected TLS error — BuildTLSConfig should have succeeded: %v", err)
	}
}

func TestConnectCluster_WithTLSErrors(t *testing.T) {
	t.Run("TLS requested but config is missing", func(t *testing.T) {
		dummyAddrs := []string{"127.0.0.1:7000"}
		client := NewClient()

		conn := types.Connection{
			Name:   "cluster-tls-missing-config",
			Host:   "127.0.0.1",
			Port:   7000,
			UseTLS: true,
			// TLSConfig is intentionally nil
		}

		err := client.ConnectCluster(dummyAddrs, conn)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		expectedErr := "TLS requested but TLS configuration is missing"
		if err.Error() != expectedErr {
			t.Errorf("expected error %q, got %q", expectedErr, err.Error())
		}
	})

	t.Run("failed to build TLS config", func(t *testing.T) {
		dummyAddrs := []string{"127.0.0.1:7000"}
		client := NewClient()

		conn := types.Connection{
			Name:   "cluster-tls-build-error",
			Host:   "127.0.0.1",
			Port:   7000,
			UseTLS: true,
			TLSConfig: &types.TLSConfig{
				// Providing invalid paths will trigger a file system read error
				CertFile: "/invalid/path/cert.pem",
				KeyFile:  "/invalid/path/key.pem",
			},
		}

		err := client.ConnectCluster(dummyAddrs, conn)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		// Use substring matching to account for OS-specific underlying errors
		if !strings.Contains(err.Error(), "failed to build TLS config") &&
			!strings.Contains(err.Error(), "failed to load TLS key pair") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// ConnectCluster — exercises the ClusterClient setup and Ping failure path.
// We pass an unreachable address so the Ping returns an error, but the
// function still walks through parseAddr, ClusterOptions construction,
// and the cluster locking branches.
// ---------------------------------------------------------------------------

func TestConnectCluster_UnreachableAddr(t *testing.T) {
	client := NewClient()

	// Address with default port and host that should not be listening.
	err := client.ConnectCluster([]string{"127.0.0.1:1"}, types.Connection{Name: "test", Host: "127.0.0.1", Port: 1, DB: 0, UseCluster: false})
	if err == nil {
		_ = client.Disconnect()
		t.Fatal("ConnectCluster expected error for unreachable addr, got nil")
	}

	// Verify the cluster fields were set even though Ping failed.
	if !client.IsCluster() {
		t.Error("IsCluster() = false after ConnectCluster, want true")
	}
	if client.host != "127.0.0.1" {
		t.Errorf("client.host = %q, want %q", client.host, "127.0.0.1")
	}
	if client.port != 1 {
		t.Errorf("client.port = %d, want %d", client.port, 1)
	}

	// Disconnect should clean up the cluster client.
	if err := client.Disconnect(); err != nil {
		t.Logf("Disconnect after failed ConnectCluster: %v", err)
	}
}

func TestConnectCluster_EmptyAddrs(t *testing.T) {
	client := NewClient()

	// No addresses — should still construct the cluster client and fail on Ping.
	err := client.ConnectCluster([]string{}, types.Connection{Name: "test", Host: "127.0.0.1", Port: 6379, DB: 0, UseCluster: false})
	if err == nil {
		_ = client.Disconnect()
		t.Fatal("ConnectCluster expected error for empty addrs, got nil")
	}

	// Default seed host/port should be used.
	if client.host != "127.0.0.1" {
		t.Errorf("client.host = %q, want %q", client.host, "127.0.0.1")
	}
	if client.port != 6379 {
		t.Errorf("client.port = %d, want %d", client.port, 6379)
	}

	_ = client.Disconnect()
}

// ---------------------------------------------------------------------------
// Disconnect — exercise the pubsub field close branch. The pubsub field is
// only ever set externally (no production setter), so we set it directly
// from the test in the same package.
// ---------------------------------------------------------------------------

func TestDisconnect_WithPubsubField(t *testing.T) {
	client, _ := setupTestClient(t)

	// Subscribe via the underlying client and store the pubsub on the
	// Client struct so disconnectLocked exercises the c.pubsub != nil branch.
	// We deliberately don't Receive the confirmation: go-redis 9.20.1 peeks a
	// fixed 36-byte push-frame window, which blocks forever on the short frame
	// miniredis returns for a brief channel name.
	client.pubsub = client.client.Subscribe(client.ctx, "fake")

	if err := client.Disconnect(); err != nil {
		t.Fatalf("Disconnect error: %v", err)
	}
	if client.pubsub != nil {
		t.Error("pubsub should be nil after Disconnect")
	}
}

// ---------------------------------------------------------------------------
// disconnectLocked — exercise cluster.Close branch via failed ConnectCluster.
// After ConnectCluster (even though Ping fails) the cluster client is set
// and Disconnect should hit the cluster Close path.
// ---------------------------------------------------------------------------

func TestDisconnect_ClusterCloseBranch(t *testing.T) {
	client := NewClient()

	// Use unreachable addr — Ping fails but cluster client is still attached.
	err := client.ConnectCluster([]string{"127.0.0.1:1"}, types.Connection{Name: "test", Host: "127.0.0.1", Port: 6379, DB: 0, UseCluster: false})
	if err == nil {
		_ = client.Disconnect()
		t.Fatal("ConnectCluster expected error, got nil")
	}

	if client.cluster == nil {
		t.Fatal("cluster client should be set after ConnectCluster even on Ping failure")
	}

	if err := client.Disconnect(); err != nil {
		t.Logf("Disconnect on failed cluster: %v", err)
	}
	if client.cluster != nil {
		t.Error("cluster should be nil after Disconnect")
	}
	if client.isCluster {
		t.Error("isCluster should be false after Disconnect")
	}
}

// ---------------------------------------------------------------------------
// ConnectCluster — invalid address (no port separator) drives the Dialer's
// SplitHostPort error path.
// ---------------------------------------------------------------------------

func TestConnectCluster_DialerSplitHostPortError(t *testing.T) {
	client := NewClient()
	t.Cleanup(func() { _ = client.Disconnect() })

	err := client.ConnectCluster([]string{"no-port"}, types.Connection{Name: "test", Host: "127.0.0.1", Port: 6379, DB: 0, UseCluster: false})
	if err == nil {
		t.Fatal("expected error from ConnectCluster with addr lacking a port")
	}
}

func TestDisconnect_AfterSubscribeKeyspace(t *testing.T) {
	client, _ := setupTestClient(t)

	handler := func(evt types.KeyspaceEvent) {}
	if err := client.SubscribeKeyspace("*", handler); err != nil {
		t.Fatalf("SubscribeKeyspace error: %v", err)
	}

	// Verify the keyspace pubsub was set up.
	if client.keyspacePS == nil {
		t.Fatal("keyspacePS should not be nil after SubscribeKeyspace")
	}
	if client.cancelKeyspace == nil {
		t.Fatal("cancelKeyspace should not be nil after SubscribeKeyspace")
	}

	// Disconnect should clean up keyspacePS and cancelKeyspace.
	if err := client.Disconnect(); err != nil {
		t.Fatalf("Disconnect error: %v", err)
	}
	if client.keyspacePS != nil {
		t.Error("keyspacePS should be nil after Disconnect")
	}
	if client.cancelKeyspace != nil {
		t.Error("cancelKeyspace should be nil after Disconnect")
	}
}

// ---------------------------------------------------------------------------
// SelectDB — SELECT command error path. Use the fake server to make SELECT
// return an error.
// ---------------------------------------------------------------------------

func TestSelectDB_SelectError(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		if argv[0] == "SELECT" {
			return "-ERR injected\r\n"
		}
		return ""
	})
	host, port := srv.addr()
	c := NewClient()
	if err := c.Connect(types.Connection{Name: "test", Host: host, Port: port, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Disconnect() })

	if err := c.SelectDB(2); err == nil {
		t.Error("expected error from SelectDB when SELECT errors")
	}
}
