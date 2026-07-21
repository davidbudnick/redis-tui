package testutil

import (
	"testing"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		str      string
		expected bool
	}{
		{"wildcard matches all", "*", "anything", true},
		{"prefix wildcard match", "user:*", "user:123", true},
		{"prefix wildcard no match", "user:*", "session:123", false},
		{"exact match", "mykey", "mykey", true},
		{"exact no match", "mykey", "otherkey", false},
		{"wildcard matches empty suffix", "user:*", "user:", true},
		{"empty pattern no match", "", "something", false},
		{"empty pattern empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchPattern(tt.pattern, tt.str)
			if got != tt.expected {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.pattern, tt.str, got, tt.expected)
			}
		})
	}
}

func TestMockRedisClient_Disconnected(t *testing.T) {
	m := NewMockRedisClient()
	// Not connected — operations should fail

	t.Run("GetValue fails when disconnected", func(t *testing.T) {
		_, err := m.GetValue("key")
		if err != ErrMockNotConnected {
			t.Errorf("expected ErrMockNotConnected, got %v", err)
		}
	})

	t.Run("ScanKeys fails when disconnected", func(t *testing.T) {
		_, _, err := m.ScanKeys("*", 0, 10)
		if err != ErrMockNotConnected {
			t.Errorf("expected ErrMockNotConnected, got %v", err)
		}
	})

	t.Run("DeleteKey fails when disconnected", func(t *testing.T) {
		err := m.DeleteKey("key")
		if err != ErrMockNotConnected {
			t.Errorf("expected ErrMockNotConnected, got %v", err)
		}
	})
}

func TestMockRedisClient_Connected(t *testing.T) {
	m := NewMockRedisClient()
	if err := m.Connect(types.Connection{Name: "test", Host: "localhost", Port: 6379, Password: "", DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	t.Run("SetKey and GetValue", func(t *testing.T) {
		val := types.RedisValue{Type: types.KeyTypeString, StringValue: "hello"}
		m.SetKey("mykey", val, types.KeyTypeString, 0)

		got, err := m.GetValue("mykey")
		if err != nil {
			t.Fatalf("GetValue error: %v", err)
		}
		if got.StringValue != "hello" {
			t.Errorf("StringValue = %q, want %q", got.StringValue, "hello")
		}
	})

	t.Run("GetValuePreview returns stored value", func(t *testing.T) {
		val := types.RedisValue{Type: types.KeyTypeString, StringValue: "preview"}
		m.SetKey("pvkey", val, types.KeyTypeString, 0)

		got, err := m.GetValuePreview("pvkey")
		if err != nil {
			t.Fatalf("GetValuePreview error: %v", err)
		}
		if got.StringValue != "preview" {
			t.Errorf("StringValue = %q, want %q", got.StringValue, "preview")
		}
	})

	t.Run("ScanKeys with pattern", func(t *testing.T) {
		m.SetKey("user:1", types.RedisValue{}, types.KeyTypeString, 0)
		m.SetKey("user:2", types.RedisValue{}, types.KeyTypeString, 0)
		m.SetKey("session:1", types.RedisValue{}, types.KeyTypeString, 0)

		keys, _, err := m.ScanKeys("user:*", 0, 10)
		if err != nil {
			t.Fatalf("ScanKeys error: %v", err)
		}
		count := 0
		for _, k := range keys {
			if k.Key == "user:1" || k.Key == "user:2" {
				count++
			}
		}
		if count != 2 {
			t.Errorf("expected 2 user keys, got %d (total keys: %d)", count, len(keys))
		}
	})

	t.Run("DeleteKey removes key", func(t *testing.T) {
		m.SetKey("toDelete", types.RedisValue{}, types.KeyTypeString, 0)
		if err := m.DeleteKey("toDelete"); err != nil {
			t.Fatalf("DeleteKey error: %v", err)
		}
		_, err := m.GetValue("toDelete")
		if err == nil {
			t.Error("expected error after deleting key")
		}
	})

	t.Run("GetTotalKeys returns count", func(t *testing.T) {
		total := m.GetTotalKeys()
		if total <= 0 {
			t.Errorf("expected positive total keys, got %d", total)
		}
	})
}

func TestMockRedisClient_Reset(t *testing.T) {
	m := NewMockRedisClient()
	_ = m.Connect(types.Connection{Name: "test", Host: "localhost", Port: 6379, Password: "", DB: 0, UseCluster: false})
	m.SetKey("key1", types.RedisValue{}, types.KeyTypeString, 0)

	m.Reset()

	if m.IsConnected() {
		t.Error("expected disconnected after Reset")
	}
	if m.GetTotalKeys() != 0 {
		t.Errorf("expected 0 keys after Reset, got %d", m.GetTotalKeys())
	}
}

func TestMockRedisClient_ConfigurableErrors(t *testing.T) {
	t.Run("ConnectError", func(t *testing.T) {
		m := NewMockRedisClient()
		m.ConnectError = ErrMockNotConnected
		err := m.Connect(types.Connection{Name: "test", Host: "localhost", Port: 6379, Password: "", DB: 0, UseCluster: false})
		if err != ErrMockNotConnected {
			t.Errorf("expected ErrMockNotConnected, got %v", err)
		}
	})

	t.Run("DisconnectError", func(t *testing.T) {
		m := NewMockRedisClient()
		_ = m.Connect(types.Connection{Name: "test", Host: "localhost", Port: 6379, Password: "", DB: 0, UseCluster: false})
		m.DisconnectError = ErrMockNotConnected
		err := m.Disconnect()
		if err != ErrMockNotConnected {
			t.Errorf("expected ErrMockNotConnected, got %v", err)
		}
	})

	t.Run("ScanError", func(t *testing.T) {
		m := NewMockRedisClient()
		_ = m.Connect(types.Connection{Name: "test", Host: "localhost", Port: 6379, Password: "", DB: 0, UseCluster: false})
		m.ScanError = ErrMockNotConnected
		_, _, err := m.ScanKeys("*", 0, 10)
		if err != ErrMockNotConnected {
			t.Errorf("expected ErrMockNotConnected, got %v", err)
		}
	})

	t.Run("GetError", func(t *testing.T) {
		m := NewMockRedisClient()
		_ = m.Connect(types.Connection{Name: "test", Host: "localhost", Port: 6379, Password: "", DB: 0, UseCluster: false})
		m.SetKey("key", types.RedisValue{}, types.KeyTypeString, 0)
		m.GetError = ErrMockNotConnected
		_, err := m.GetValue("key")
		if err != ErrMockNotConnected {
			t.Errorf("expected ErrMockNotConnected, got %v", err)
		}
	})

	t.Run("DeleteError", func(t *testing.T) {
		m := NewMockRedisClient()
		_ = m.Connect(types.Connection{Name: "test", Host: "localhost", Port: 6379, Password: "", DB: 0, UseCluster: false})
		m.DeleteError = ErrMockNotConnected
		err := m.DeleteKey("key")
		if err != ErrMockNotConnected {
			t.Errorf("expected ErrMockNotConnected, got %v", err)
		}
	})
}

func TestMockRedisClient_DisconnectSuccess(t *testing.T) {
	m := NewMockRedisClient()
	if err := m.Connect(types.Connection{Name: "test", Host: "localhost", Port: 6379, Password: "", DB: 0, UseCluster: false}); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	if !m.IsConnected() {
		t.Fatal("expected connected after Connect")
	}
	if err := m.Disconnect(); err != nil {
		t.Fatalf("Disconnect failed: %v", err)
	}
	if m.IsConnected() {
		t.Error("expected disconnected after Disconnect")
	}
}

func TestMockRedisClient_TTLStored(t *testing.T) {
	m := NewMockRedisClient()
	_ = m.Connect(types.Connection{Name: "test", Host: "localhost", Port: 6379, Password: "", DB: 0, UseCluster: false})
	m.SetKey("ttlkey", types.RedisValue{}, types.KeyTypeString, 5*time.Second)

	keys, _, err := m.ScanKeys("*", 0, 10)
	if err != nil {
		t.Fatalf("ScanKeys error: %v", err)
	}
	for _, k := range keys {
		if k.Key == "ttlkey" {
			if k.TTL != 5*time.Second {
				t.Errorf("TTL = %v, want %v", k.TTL, 5*time.Second)
			}
			return
		}
	}
	t.Error("ttlkey not found in scan results")
}
