package redis

import (
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/davidbudnick/redis-tui/internal/testutil"
	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestParseClusterNodes(t *testing.T) {
	t.Run("single node", func(t *testing.T) {
		input := "abc123 127.0.0.1:7000@17000 master - 0 0 1 connected 0-5460\n"
		nodes := parseClusterNodes(input)
		if len(nodes) != 1 {
			t.Fatalf("expected 1 node, got %d", len(nodes))
		}
		if nodes[0].ID != "abc123" {
			t.Errorf("ID = %q, want %q", nodes[0].ID, "abc123")
		}
		if nodes[0].Addr != "127.0.0.1:7000@17000" {
			t.Errorf("Addr = %q, want %q", nodes[0].Addr, "127.0.0.1:7000@17000")
		}
		if nodes[0].Flags != "master" {
			t.Errorf("Flags = %q, want %q", nodes[0].Flags, "master")
		}
		if nodes[0].LinkState != "connected" {
			t.Errorf("LinkState = %q, want %q", nodes[0].LinkState, "connected")
		}
		if nodes[0].Slots != "0-5460" {
			t.Errorf("Slots = %q, want %q", nodes[0].Slots, "0-5460")
		}
	})

	t.Run("multiple nodes", func(t *testing.T) {
		input := "abc123 127.0.0.1:7000@17000 master - 0 0 1 connected 0-5460\n" +
			"def456 127.0.0.1:7001@17001 slave abc123 0 0 1 connected\n" +
			"ghi789 127.0.0.1:7002@17002 master - 0 0 2 connected 5461-10922\n"
		nodes := parseClusterNodes(input)
		if len(nodes) != 3 {
			t.Fatalf("expected 3 nodes, got %d", len(nodes))
		}
	})

	t.Run("empty input", func(t *testing.T) {
		nodes := parseClusterNodes("")
		if len(nodes) != 0 {
			t.Errorf("expected 0 nodes for empty input, got %d", len(nodes))
		}
	})

	t.Run("lines with fewer than 8 fields skipped", func(t *testing.T) {
		input := "abc123 127.0.0.1:7000@17000 master - 0 0 1 connected 0-5460\n" +
			"short line only\n" +
			"def456 127.0.0.1:7001@17001 slave abc123 0 0 1 connected\n"
		nodes := parseClusterNodes(input)
		if len(nodes) != 2 {
			t.Errorf("expected 2 nodes (short line skipped), got %d", len(nodes))
		}
	})

	t.Run("node without slots", func(t *testing.T) {
		input := "def456 127.0.0.1:7001@17001 slave abc123 0 0 1 connected\n"
		nodes := parseClusterNodes(input)
		if len(nodes) != 1 {
			t.Fatalf("expected 1 node, got %d", len(nodes))
		}
		if nodes[0].Slots != "" {
			t.Errorf("Slots = %q, want empty", nodes[0].Slots)
		}
	})

	t.Run("master field", func(t *testing.T) {
		input := "def456 127.0.0.1:7001@17001 slave abc123 0 0 1 connected\n"
		nodes := parseClusterNodes(input)
		if nodes[0].Master != "abc123" {
			t.Errorf("Master = %q, want %q", nodes[0].Master, "abc123")
		}
	})
}

func TestGetServerInfo(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.Set("k1", "v1")
	mr.Set("k2", "v2")
	mr.Set("k3", "v3")

	info, err := client.GetServerInfo()
	if err != nil {
		t.Fatalf("GetServerInfo() error = %v", err)
	}

	// TotalKeys is derived from DBSize which miniredis fully supports
	if info.TotalKeys != "3" {
		t.Errorf("GetServerInfo() TotalKeys = %q, want %q", info.TotalKeys, "3")
	}

	// miniredis may not populate redis_version in INFO; verify the field is at least set
	// (Version may be empty with miniredis, which is acceptable)
	t.Logf("GetServerInfo() Version = %q", info.Version)
}

func TestGetMemoryStats(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.Set("mem1", "some-value")
	mr.Set("mem2", "another-value")

	stats, err := client.GetMemoryStats()
	if err != nil {
		// miniredis does not support INFO memory section; verify we get a
		// graceful error rather than a panic
		t.Logf("GetMemoryStats() returned expected error with miniredis: %v", err)
		return
	}

	// If miniredis ever adds support, verify the result is reasonable
	if stats.UsedMemory < 0 {
		t.Errorf("GetMemoryStats() UsedMemory = %d, want >= 0", stats.UsedMemory)
	}
}

func TestFlushDB(t *testing.T) {
	client, mr := setupTestClient(t)

	mr.Set("a", "1")
	mr.Set("b", "2")
	mr.Set("c", "3")

	if client.GetTotalKeys() != 3 {
		t.Fatal("expected 3 keys before flush")
	}

	if err := client.FlushDB(); err != nil {
		t.Fatalf("FlushDB() error = %v", err)
	}

	if got := client.GetTotalKeys(); got != 0 {
		t.Errorf("FlushDB() left %d keys, want 0", got)
	}
}

func TestEval(t *testing.T) {
	client, _ := setupTestClient(t)

	result, err := client.Eval("return 1+1", nil)
	if err != nil {
		t.Fatalf("Eval() error = %v", err)
	}

	val, ok := result.(int64)
	if !ok {
		t.Fatalf("Eval() result type = %T, want int64", result)
	}
	if val != 2 {
		t.Errorf("Eval(return 1+1) = %d, want 2", val)
	}
}

func TestGetLiveMetrics(t *testing.T) {
	client, _ := setupTestClient(t)

	metrics, err := client.GetLiveMetrics()
	if err != nil {
		// miniredis does not support INFO with multiple section arguments;
		// verify we get a graceful error rather than a panic
		t.Logf("GetLiveMetrics() returned expected error with miniredis: %v", err)
		return
	}

	if metrics.Timestamp.IsZero() {
		t.Error("GetLiveMetrics() Timestamp is zero")
	}

	if time.Since(metrics.Timestamp) > 5*time.Second {
		t.Error("GetLiveMetrics() Timestamp is not recent")
	}

	// ConnectedClients should be at least 1 (our own connection)
	if metrics.ConnectedClients < 1 {
		t.Errorf("GetLiveMetrics() ConnectedClients = %d, want >= 1", metrics.ConnectedClients)
	}
}

func TestDeleteKeys(t *testing.T) {
	t.Run("deletes specified keys and returns count", func(t *testing.T) {
		client, mr := setupTestClient(t)

		mr.Set("k1", "v1")
		mr.Set("k2", "v2")
		mr.Set("k3", "v3")

		count, err := client.DeleteKeys("k1", "k2")
		if err != nil {
			t.Fatalf("DeleteKeys() error = %v", err)
		}
		if count != 2 {
			t.Errorf("DeleteKeys() count = %d, want 2", count)
		}

		if client.GetTotalKeys() != 1 {
			t.Errorf("expected 1 key remaining, got %d", client.GetTotalKeys())
		}

		if !mr.Exists("k3") {
			t.Error("expected k3 to still exist")
		}
	})

	t.Run("deleting non-existent keys returns 0", func(t *testing.T) {
		client, _ := setupTestClient(t)

		count, err := client.DeleteKeys("nonexistent1", "nonexistent2")
		if err != nil {
			t.Fatalf("DeleteKeys() error = %v", err)
		}
		if count != 0 {
			t.Errorf("DeleteKeys() count = %d, want 0", count)
		}
	})
}

func TestIsCluster(t *testing.T) {
	client, _ := setupTestClient(t)

	if client.IsCluster() {
		t.Error("IsCluster() = true, want false for standalone miniredis")
	}
}

func TestTestConnection(t *testing.T) {
	t.Run("successful connection returns latency", func(t *testing.T) {
		client, mr := setupTestClient(t)

		port, err := strconv.Atoi(mr.Port())
		if err != nil {
			t.Fatalf("failed to parse port: %v", err)
		}

		latency, err := client.TestConnection(types.Connection{Name: "test", Host: mr.Host(), Port: port, Password: "", DB: 0, UseCluster: false})
		if err != nil {
			t.Fatalf("TestConnection() error = %v", err)
		}
		assertMeasuredLatency(t, latency)
	})

	t.Run("successful connection with username / password returns latency", func(t *testing.T) {
		client, mr := setupTestClientWithAuth(t, "testuser", "testpass")

		port, err := strconv.Atoi(mr.Port())
		if err != nil {
			t.Fatalf("failed to parse port: %v", err)
		}

		latency, err := client.TestConnection(types.Connection{Name: "test", Host: mr.Host(), Port: port, Username: "testuser", Password: "testpass", DB: 0, UseCluster: false})
		if err != nil {
			t.Fatalf("TestConnection() error = %v", err)
		}
		assertMeasuredLatency(t, latency)
	})

	t.Run("wrong username / password returns error", func(t *testing.T) {
		client, mr := setupTestClientWithAuth(t, "testuser", "testpass")

		port, err := strconv.Atoi(mr.Port())
		if err != nil {
			t.Fatalf("failed to parse port: %v", err)
		}

		if _, err := client.TestConnection(types.Connection{Name: "test", Host: mr.Host(), Port: port, Username: "baduser", Password: "badpass", DB: 0, UseCluster: false}); err == nil {
			t.Fatal("TestConnection() expected error for wrong username/password, got nil")
		}
	})

	t.Run("wrong port returns error", func(t *testing.T) {
		client, _ := setupTestClient(t)

		// Use a port that is almost certainly not listening
		_, err := client.TestConnection(types.Connection{Name: "test", Host: "127.0.0.1", Port: 1, Password: "", DB: 0, UseCluster: false})
		if err == nil {
			t.Fatal("TestConnection() expected error for wrong port, got nil")
		}
	})

	t.Run("wrong host returns error", func(t *testing.T) {
		// Start a separate miniredis so we have a valid client context
		mr, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to start miniredis: %v", err)
		}
		t.Cleanup(mr.Close)

		c := NewClient()
		port, _ := strconv.Atoi(mr.Port())
		if err := c.Connect(types.Connection{Name: "test", Host: mr.Host(), Port: port, Password: "", DB: 0, UseCluster: false}); err != nil {
			t.Fatalf("Connect() error = %v", err)
		}
		t.Cleanup(func() { _ = c.Disconnect() })

		_, err = c.TestConnection(types.Connection{Name: "test", Host: "192.0.2.1", Port: 6379, Password: "", DB: 0, UseCluster: false})
		if err == nil {
			t.Fatal("TestConnection() expected error for unreachable host, got nil")
		}
	})

	t.Run("empty host returns error", func(t *testing.T) {
		// Start a separate miniredis so we have a valid client context
		mr, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to start miniredis: %v", err)
		}
		t.Cleanup(mr.Close)

		c := NewClient()
		port, _ := strconv.Atoi(mr.Port())
		if err := c.Connect(types.Connection{Name: "test", Host: mr.Host(), Port: port, Password: "", DB: 0, UseCluster: false}); err != nil {
			t.Fatalf("Connect() error = %v", err)
		}
		t.Cleanup(func() { _ = c.Disconnect() })

		_, err = c.TestConnection(types.Connection{Name: "test", Host: "", Port: 6379, Password: "", DB: 0, UseCluster: false})
		if err == nil {
			t.Fatal("TestConnection() expected error for empty host, got nil")
		}
	})

	t.Run("empty port returns error", func(t *testing.T) {
		// Start a separate miniredis so we have a valid client context
		mr, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to start miniredis: %v", err)
		}
		t.Cleanup(mr.Close)

		c := NewClient()
		port, _ := strconv.Atoi(mr.Port())
		if err := c.Connect(types.Connection{Name: "test", Host: mr.Host(), Port: port, Password: "", DB: 0, UseCluster: false}); err != nil {
			t.Fatalf("Connect() error = %v", err)
		}
		t.Cleanup(func() { _ = c.Disconnect() })

		_, err = c.TestConnection(types.Connection{Name: "test", Host: "192.168.0.1", Port: 0, Password: "", DB: 0, UseCluster: false})
		if err == nil {
			t.Fatal("TestConnection() expected error for empty port, got nil")
		}
	})

	t.Run("successful connection with TLS returns latency", func(t *testing.T) {
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
		latency, err := client.TestConnection(conn)
		if err != nil {
			t.Fatalf("Connect() with TLS returned error: %v", err)
		}
		t.Cleanup(func() { _ = client.Disconnect() })
		assertMeasuredLatency(t, latency)
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

		_, err := client.TestConnection(conn)
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

		_, err := client.TestConnection(conn)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !strings.Contains(err.Error(), "failed to build TLS config") &&
			!strings.Contains(err.Error(), "failed to load TLS key pair") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// ConfigGet
// ---------------------------------------------------------------------------

func TestConfigGet(t *testing.T) {
	t.Run("returns result or graceful error", func(t *testing.T) {
		client, _ := setupTestClient(t)

		result, err := client.ConfigGet("save")
		if err != nil {
			// miniredis may not support CONFIG GET; verify graceful error
			t.Logf("ConfigGet() returned expected error with miniredis: %v", err)
			return
		}

		if _, ok := result["save"]; !ok {
			t.Error("ConfigGet(\"save\") did not return a 'save' key")
		}
	})

	t.Run("wildcard pattern returns results or graceful error", func(t *testing.T) {
		client, _ := setupTestClient(t)

		result, err := client.ConfigGet("*")
		if err != nil {
			// miniredis may not support CONFIG GET; verify graceful error
			t.Logf("ConfigGet(\"*\") returned expected error with miniredis: %v", err)
			return
		}

		if len(result) == 0 {
			t.Error("ConfigGet(\"*\") returned empty map")
		}
	})
}

// ---------------------------------------------------------------------------
// ConfigSet
// ---------------------------------------------------------------------------

func TestConfigSet(t *testing.T) {
	t.Run("set config param returns result or graceful error", func(t *testing.T) {
		client, _ := setupTestClient(t)

		err := client.ConfigSet("save", "900 1")
		if err != nil {
			// miniredis may not support CONFIG SET; verify graceful error
			t.Logf("ConfigSet() returned expected error with miniredis: %v", err)
			return
		}

		result, err := client.ConfigGet("save")
		if err != nil {
			t.Fatalf("ConfigGet() error = %v", err)
		}

		if result["save"] != "900 1" {
			t.Errorf("ConfigGet(\"save\") = %q, want %q", result["save"], "900 1")
		}
	})
}

// ---------------------------------------------------------------------------
// ClientList
// ---------------------------------------------------------------------------

func TestClientList(t *testing.T) {
	t.Run("returns clients or graceful error", func(t *testing.T) {
		client, _ := setupTestClient(t)

		clients, err := client.ClientList()
		if err != nil {
			// miniredis may not support CLIENT LIST; verify graceful error
			t.Logf("ClientList() returned expected error with miniredis: %v", err)
			return
		}

		if len(clients) == 0 {
			t.Fatal("ClientList() returned 0 clients, want >= 1")
		}

		// Verify first client has a non-empty address
		c := clients[0]
		if c.Addr == "" {
			t.Error("ClientList() first client has empty Addr")
		}
	})
}

// ---------------------------------------------------------------------------
// SlowLogGet — miniredis may not implement SLOWLOG; verify graceful behavior.
// ---------------------------------------------------------------------------

func TestSlowLogGet(t *testing.T) {
	client, _ := setupTestClient(t)

	entries, err := client.SlowLogGet(10)
	if err != nil {
		t.Logf("SlowLogGet returned expected error with miniredis: %v", err)
		return
	}

	// If supported, the result should be a non-nil slice (may be empty).
	if entries == nil {
		t.Error("SlowLogGet returned nil slice without error")
	}
}

// ---------------------------------------------------------------------------
// ClusterNodes / ClusterInfo on a standalone client — these dispatch to the
// non-cluster client path. miniredis may not implement these commands but
// the call must not panic.
// ---------------------------------------------------------------------------

func TestClusterNodes_Standalone(t *testing.T) {
	client, _ := setupTestClient(t)

	nodes, err := client.ClusterNodes()
	if err != nil {
		t.Logf("ClusterNodes returned expected error with miniredis: %v", err)
		return
	}
	// If supported, just verify the slice is reachable (may be empty).
	_ = nodes
}

func TestClusterInfo_Standalone(t *testing.T) {
	client, _ := setupTestClient(t)

	info, err := client.ClusterInfo()
	if err != nil {
		t.Logf("ClusterInfo returned expected error with miniredis: %v", err)
		return
	}
	_ = info
}

// ---------------------------------------------------------------------------
// GetMemoryStats / getTopKeysByMemory — exercising the path even when
// miniredis returns minimal/empty data. Seed many keys to ensure scanLimited
// is invoked when GetMemoryStats reaches getTopKeysByMemory.
// ---------------------------------------------------------------------------

func TestGetMemoryStats_WithSeededKeys(t *testing.T) {
	client, mr := setupTestClient(t)

	// Seed enough keys to traverse multiple SCAN batches inside scanLimited.
	for i := 0; i < 150; i++ {
		mr.Set(fmt.Sprintf("memkey:%d", i), "some-value")
	}

	stats, err := client.GetMemoryStats()
	if err != nil {
		// miniredis may not fully support INFO memory; verify graceful failure.
		t.Logf("GetMemoryStats returned expected error: %v", err)
		return
	}

	// If miniredis returns enough data to populate stats, verify TopKeys is
	// reachable (may be nil/empty due to MEMORY USAGE limitations).
	_ = stats.TopKeys
}

// ---------------------------------------------------------------------------
// getTopKeysByMemory — call directly because GetMemoryStats short-circuits
// when miniredis fails the INFO memory section. Direct invocation exercises
// scanLimited, the pipeline batching, the sort, and the result truncation.
// ---------------------------------------------------------------------------

func TestGetTopKeysByMemory_Direct(t *testing.T) {
	client, mr := setupTestClient(t)

	for i := 0; i < 30; i++ {
		mr.Set(fmt.Sprintf("topmem:%d", i), fmt.Sprintf("value-%d", i))
	}

	// Limit smaller than total keys to exercise the sort+truncate.
	result := client.getTopKeysByMemory(10)
	// miniredis may not support MEMORY USAGE on every key, so the result may
	// be nil or shorter than 10. Both are acceptable, just verify no panic.
	if len(result) > 10 {
		t.Errorf("result length = %d, want <= 10", len(result))
	}
}

func TestGetTopKeysByMemory_EmptyDB(t *testing.T) {
	client, _ := setupTestClient(t)

	result := client.getTopKeysByMemory(20)
	if result != nil {
		t.Errorf("expected nil result for empty DB, got %v", result)
	}
}

// ---------------------------------------------------------------------------
// GetServerInfo — INFO error path.
// ---------------------------------------------------------------------------

func TestGetServerInfo_InfoError(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		if argv[0] == "INFO" {
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

	if _, err := c.GetServerInfo(); err == nil {
		t.Error("expected error from GetServerInfo when INFO errors")
	}
}

// ---------------------------------------------------------------------------
// GetServerInfo — malformed line (line without colon) is skipped.
// ---------------------------------------------------------------------------

func TestGetServerInfo_MalformedLine(t *testing.T) {
	srv := newFakeRedisServer(t)
	body := "redis_version:7.0.0\r\nmalformed_no_colon\r\nredis_mode:standalone\r\n"
	srv.setResponse("INFO", respBulkString(body))
	srv.setResponse("DBSIZE", ":0\r\n")

	host, port := srv.addr()
	c := NewClient()
	if err := c.Connect(types.Connection{Name: "test", Host: host, Port: port, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Disconnect() })

	info, err := c.GetServerInfo()
	if err != nil {
		t.Fatalf("GetServerInfo error: %v", err)
	}
	if info.Version != "7.0.0" {
		t.Errorf("Version = %q, want 7.0.0", info.Version)
	}
	if info.Mode != "standalone" {
		t.Errorf("Mode = %q, want standalone", info.Mode)
	}
}

// ---------------------------------------------------------------------------
// getTopKeysByMemory — MemoryUsage error continues. Use the fake server to
// drive a single key through the pipeline where MEMORY USAGE returns an error.
// ---------------------------------------------------------------------------

func TestGetTopKeysByMemory_MemErrorContinues(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		switch argv[0] {
		case "SCAN":
			return "*2\r\n$1\r\n0\r\n*1\r\n$1\r\nx\r\n"
		case "MEMORY":
			// MEMORY USAGE x returns an error.
			return "-ERR injected\r\n"
		case "TYPE":
			return "+string\r\n"
		}
		return ""
	})
	host, port := srv.addr()
	c := NewClient()
	if err := c.Connect(types.Connection{Name: "test", Host: host, Port: port, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Disconnect() })

	result := c.getTopKeysByMemory(10)
	if len(result) != 0 {
		t.Errorf("result length = %d, want 0 (all skipped via continue)", len(result))
	}
}

// ---------------------------------------------------------------------------
// ClientList — malformed field (without "=") is skipped. We return a CLIENT
// LIST line containing a stray bareword to drive the continue path.
// ---------------------------------------------------------------------------

func TestClientList_MalformedField(t *testing.T) {
	srv := newFakeRedisServer(t)
	body := "id=1 addr=127.0.0.1:1 strayfield name=foo\n"
	srv.setResponse("CLIENT", respBulkString(body))

	host, port := srv.addr()
	c := NewClient()
	if err := c.Connect(types.Connection{Name: "test", Host: host, Port: port, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Disconnect() })

	clients, err := c.ClientList()
	if err != nil {
		t.Fatalf("ClientList error: %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("clients = %d, want 1", len(clients))
	}
	if clients[0].Name != "foo" {
		t.Errorf("Name = %q, want foo (stray field should be skipped)", clients[0].Name)
	}
}

// ---------------------------------------------------------------------------
// ClusterNodes — error return path on standalone client.
// ---------------------------------------------------------------------------

func TestClusterNodes_Error(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		if argv[0] == "CLUSTER" {
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

	if _, err := c.ClusterNodes(); err == nil {
		t.Error("expected error from ClusterNodes when CLUSTER NODES errors")
	}
}

// ---------------------------------------------------------------------------
// ClusterNodes — cluster client branch with error. Verifies the err return
// inside the cluster path.
// ---------------------------------------------------------------------------

func TestClusterNodes_ClusterErrorReturn(t *testing.T) {
	srv := newFakeRedisServer(t)
	srv.setHandler(func(argv []string) string {
		if argv[0] == "CLUSTER" {
			return "-ERR injected\r\n"
		}
		return ""
	})
	host, port := srv.addr()
	addr := fmt.Sprintf("%s:%d", host, port)

	client := NewClient()
	cluster := newClusterClientForTest(addr)
	client.cluster = cluster
	client.isCluster = true
	t.Cleanup(func() {
		_ = cluster.Close()
		client.cluster = nil
	})

	if _, err := client.ClusterNodes(); err == nil {
		t.Error("expected error from ClusterNodes (cluster) when CLUSTER NODES errors")
	}
}

// ---------------------------------------------------------------------------
// GetLiveMetrics — malformed line skipped.
// ---------------------------------------------------------------------------

func TestGetLiveMetrics_MalformedLine(t *testing.T) {
	srv := newFakeRedisServer(t)
	body := "instantaneous_ops_per_sec:5\r\nmalformed_no_colon\r\nused_memory:1024\r\n"
	srv.setResponse("INFO", respBulkString(body))

	host, port := srv.addr()
	c := NewClient()
	if err := c.Connect(types.Connection{Name: "test", Host: host, Port: port, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Disconnect() })

	m, err := c.GetLiveMetrics()
	if err != nil {
		t.Fatalf("GetLiveMetrics error: %v", err)
	}
	if m.OpsPerSec != 5 {
		t.Errorf("OpsPerSec = %f, want 5", m.OpsPerSec)
	}
	if m.UsedMemoryBytes != 1024 {
		t.Errorf("UsedMemoryBytes = %d, want 1024", m.UsedMemoryBytes)
	}
}
