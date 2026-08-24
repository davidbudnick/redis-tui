package redis

import (
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/davidbudnick/redis-tui/internal/types"
)

func setupTestClient(t *testing.T) (*Client, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := NewClient()
	port, _ := strconv.Atoi(mr.Port())
	if err := client.Connect(types.Connection{Name: "test", Host: mr.Host(), Port: port, Password: "", DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect() })

	return client, mr
}

func setupTestClientWithAuth(t *testing.T, username string, password string) (*Client, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	mr.RequireUserAuth(username, password)

	client := NewClient()
	port, _ := strconv.Atoi(mr.Port())
	if err := client.Connect(types.Connection{Name: "test", Host: mr.Host(), Port: port, Username: username, Password: password, DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect() })

	return client, mr
}

// assertMeasuredLatency accepts 0 on Windows where the clock can miss a localhost ping.
func assertMeasuredLatency(t *testing.T, latency time.Duration) {
	t.Helper()
	if latency > 0 {
		return
	}
	if latency == 0 && runtime.GOOS == "windows" {
		return
	}
	t.Errorf("TestConnection() latency = %v, want > 0", latency)
}
