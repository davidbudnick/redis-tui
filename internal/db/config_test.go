package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestNewConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg, err := NewConfig(path)
	if err != nil {
		t.Fatalf("NewConfig failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("NewConfig returned nil")
	}

	// Check defaults
	if cfg.TreeSeparator != ":" {
		t.Errorf("TreeSeparator = %q, want \":\"", cfg.TreeSeparator)
	}
	if cfg.MaxRecentKeys != 20 {
		t.Errorf("MaxRecentKeys = %d, want 20", cfg.MaxRecentKeys)
	}
	if cfg.MaxValueHistory != 50 {
		t.Errorf("MaxValueHistory = %d, want 50", cfg.MaxValueHistory)
	}
	if cfg.WatchInterval != 1000 {
		t.Errorf("WatchInterval = %d, want 1000", cfg.WatchInterval)
	}
}

func TestNewConfig_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	nestedPath := filepath.Join(dir, "subdir", "config.json")

	_, err := NewConfig(nestedPath)
	if err != nil {
		t.Fatalf("NewConfig failed: %v", err)
	}

	// Check directory was created
	if _, err := os.Stat(filepath.Dir(nestedPath)); os.IsNotExist(err) {
		t.Error("NewConfig did not create directory")
	}
}

func TestConfig_AddConnection(t *testing.T) {
	cfg := newTestConfig(t)

	conn, err := cfg.AddConnection(types.Connection{Name: "test", Host: "localhost", Port: 6379, Username: "default", Password: "secret", DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	if conn.ID == 0 {
		t.Error("Connection ID should not be 0")
	}
	if conn.Name != "test" {
		t.Errorf("Name = %q, want \"test\"", conn.Name)
	}
	if conn.Host != "localhost" {
		t.Errorf("Host = %q, want \"localhost\"", conn.Host)
	}
	if conn.Port != 6379 {
		t.Errorf("Port = %d, want 6379", conn.Port)
	}
	if conn.Username != "default" {
		t.Errorf("Username = %q, want \"default\"", conn.Username)
	}
	if conn.Password != "secret" {
		t.Errorf("Password = %q, want \"secret\"", conn.Password)
	}
	if conn.DB != 0 {
		t.Errorf("DB = %d, want 0", conn.DB)
	}
	if conn.Created.IsZero() {
		t.Error("Created should not be zero")
	}
}

func TestConfig_AddConnection_IncrementingIDs(t *testing.T) {
	cfg := newTestConfig(t)

	conn1, err := cfg.AddConnection(types.Connection{Name: "test1", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}
	conn2, err := cfg.AddConnection(types.Connection{Name: "test2", Host: "localhost", Port: 6380, DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}
	conn3, err := cfg.AddConnection(types.Connection{Name: "test3", Host: "localhost", Port: 6381, DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	if conn2.ID <= conn1.ID {
		t.Errorf("conn2.ID (%d) should be greater than conn1.ID (%d)", conn2.ID, conn1.ID)
	}
	if conn3.ID <= conn2.ID {
		t.Errorf("conn3.ID (%d) should be greater than conn2.ID (%d)", conn3.ID, conn2.ID)
	}
}

func TestConfig_ListConnections(t *testing.T) {
	cfg := newTestConfig(t)

	// Add connections in non-alphabetical order
	_, err := cfg.AddConnection(types.Connection{Name: "zebra", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}
	_, err = cfg.AddConnection(types.Connection{Name: "alpha", Host: "localhost", Port: 6380, DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}
	_, err = cfg.AddConnection(types.Connection{Name: "beta", Host: "localhost", Port: 6381, DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	connections, errList := cfg.ListConnections()
	if errList != nil {
		t.Fatalf("ListConnections failed: %v", errList)
	}

	if len(connections) != 3 {
		t.Fatalf("Expected 3 connections, got %d", len(connections))
	}

	// Check sorted by ID (insertion order)
	if connections[0].Name != "zebra" {
		t.Errorf("First connection = %q, want \"zebra\"", connections[0].Name)
	}
	if connections[1].Name != "alpha" {
		t.Errorf("Second connection = %q, want \"alpha\"", connections[1].Name)
	}
	if connections[2].Name != "beta" {
		t.Errorf("Third connection = %q, want \"beta\"", connections[2].Name)
	}
}

func TestConfig_UpdateConnection(t *testing.T) {
	cfg := newTestConfig(t)

	conn, err := cfg.AddConnection(types.Connection{Name: "original", Host: "localhost", Port: 6379, Username: "default", Password: "old", DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}
	originalCreated := conn.Created
	conn.Name = "updated"
	conn.Host = "newhost"
	conn.Port = 6380
	conn.Username = "newuser"
	conn.Password = "new"
	conn.DB = 1

	updated, err := cfg.UpdateConnection(conn)
	if err != nil {
		t.Fatalf("UpdateConnection failed: %v", err)
	}

	if updated.Name != "updated" {
		t.Errorf("Name = %q, want \"updated\"", updated.Name)
	}
	if updated.Host != "newhost" {
		t.Errorf("Host = %q, want \"newhost\"", updated.Host)
	}
	if updated.Port != 6380 {
		t.Errorf("Port = %d, want 6380", updated.Port)
	}
	if updated.Username != "newuser" {
		t.Errorf("Username = %q, want \"newuser\"", updated.Username)
	}
	if updated.Password != "new" {
		t.Errorf("Password = %q, want \"new\"", updated.Password)
	}
	if updated.DB != 1 {
		t.Errorf("DB = %d, want 1", updated.DB)
	}
	if !updated.Created.Equal(originalCreated) {
		t.Error("Created timestamp should not change")
	}
	if updated.Updated.Before(updated.Created) {
		t.Error("Updated should not be before Created")
	}
}

func TestConfig_UpdateConnection_NotFound(t *testing.T) {
	cfg := newTestConfig(t)

	_, err := cfg.UpdateConnection(types.Connection{ID: 999, Name: "test", Host: "localhost", Port: 6379, Password: "", DB: 0, UseCluster: false})
	if !os.IsNotExist(err) {
		t.Errorf("Expected os.ErrNotExist, got %v", err)
	}
}

func TestConfig_DeleteConnection(t *testing.T) {
	cfg := newTestConfig(t)

	conn, err := cfg.AddConnection(types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	err = cfg.DeleteConnection(conn.ID)
	if err != nil {
		t.Fatalf("DeleteConnection failed: %v", err)
	}

	connections, errList := cfg.ListConnections()
	if errList != nil {
		t.Fatalf("ListConnections failed: %v", errList)
	}
	if len(connections) != 0 {
		t.Errorf("Expected 0 connections, got %d", len(connections))
	}
}

func TestConfig_DeleteConnection_NotFound(t *testing.T) {
	cfg := newTestConfig(t)

	err := cfg.DeleteConnection(999)
	if !os.IsNotExist(err) {
		t.Errorf("Expected os.ErrNotExist, got %v", err)
	}
}

func TestDefaultTemplates(t *testing.T) {
	templates := defaultTemplates()

	if len(templates) == 0 {
		t.Fatal("Expected default templates")
	}

	// Check required templates exist
	requiredNames := []string{"Session", "Cache", "Rate Limit", "Queue", "Leaderboard"}
	for _, name := range requiredNames {
		found := false
		for _, tmpl := range templates {
			if tmpl.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing required template: %q", name)
		}
	}
}

// newTestConfig creates a config for testing with a temp file
func newTestConfig(t *testing.T) *Config {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg, err := NewConfig(path)
	if err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}
	return cfg
}

// reloadConfig creates a fresh Config from the same file path, forcing a full JSON round-trip.
func reloadConfig(t *testing.T, cfg *Config) *Config {
	t.Helper()
	cfg2, err := NewConfig(cfg.path)
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	return cfg2
}

// contains checks if a string contains a substring (test helper to avoid importing strings).
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
