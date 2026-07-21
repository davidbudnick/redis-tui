package redis

import (
	"strconv"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/davidbudnick/redis-tui/internal/types"
)

// setupBenchClient mirrors setupTestClient for benchmarks.
func setupBenchClient(b *testing.B) (*Client, *miniredis.Miniredis) {
	b.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatalf("failed to start miniredis: %v", err)
	}
	b.Cleanup(mr.Close)

	client := NewClient()
	port, err := strconv.Atoi(mr.Port())
	if err != nil {
		b.Fatalf("bad port: %v", err)
	}
	if err := client.Connect(types.Connection{Name: "bench", Host: mr.Host(), Port: port}); err != nil {
		b.Fatalf("failed to connect: %v", err)
	}
	b.Cleanup(func() { _ = client.Disconnect() })

	return client, mr
}
