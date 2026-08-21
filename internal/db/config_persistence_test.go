package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
)

// --- Phase 1: Connection Persistence Round-Trip ---

func TestConfig_Persistence_AllConnectionFields(t *testing.T) {
	cfg := newTestConfig(t)

	conn, err := cfg.AddConnection(types.Connection{Name: "prod-redis", Host: "redis.example.com", Username: "default", Port: 6380, VaultPath: "secret/data/redis/prod", VaultUserKey: "credentials.redis.username", VaultPasswordKey: "credentials.redis.password", DB: 2, UseCluster: true})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	cfg2 := reloadConfig(t, cfg)
	connections, err := cfg2.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections failed: %v", err)
	}
	if len(connections) != 1 {
		t.Fatalf("expected 1 connection after reload, got %d", len(connections))
	}

	got := connections[0]
	if got.ID != conn.ID {
		t.Errorf("ID = %d, want %d", got.ID, conn.ID)
	}
	if got.Name != "prod-redis" {
		t.Errorf("Name = %q, want %q", got.Name, "prod-redis")
	}
	if got.Host != "redis.example.com" {
		t.Errorf("Host = %q, want %q", got.Host, "redis.example.com")
	}
	if got.Port != 6380 {
		t.Errorf("Port = %d, want %d", got.Port, 6380)
	}
	if got.Username != "default" {
		t.Errorf("Username = %q, want %q", got.Username, "default")
	}
	if got.VaultPath != "secret/data/redis/prod" {
		t.Errorf("VaultPath = %q, want %q", got.VaultPath, "secret/data/redis/prod")
	}
	if got.VaultUserKey != "credentials.redis.username" {
		t.Errorf("VaultUserKey = %q, want %q", got.VaultUserKey, "credentials.redis.username")
	}
	if got.VaultPasswordKey != "credentials.redis.password" {
		t.Errorf("VaultPasswordKey = %q, want %q", got.VaultPasswordKey, "credentials.redis.password")
	}
	if got.DB != 2 {
		t.Errorf("DB = %d, want %d", got.DB, 2)
	}
	if got.UseCluster != true {
		t.Errorf("UseCluster = %v, want true", got.UseCluster)
	}
	if got.Created.IsZero() {
		t.Error("Created should not be zero after reload")
	}
	if got.Updated.IsZero() {
		t.Error("Updated should not be zero after reload")
	}
}

func TestConfig_Persistence_PasswordStripping(t *testing.T) {
	cfg := newTestConfig(t)

	conn, err := cfg.AddConnection(types.Connection{Name: "secure", Host: "localhost", Port: 6379, Password: "s3cr3t_p@ss", DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	// In-memory connection should have the password
	if conn.Password != "s3cr3t_p@ss" {
		t.Errorf("in-memory password = %q, want %q", conn.Password, "s3cr3t_p@ss")
	}

	// Raw JSON file should NOT contain the password
	data, err := os.ReadFile(cfg.path)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}
	if string(data) == "" {
		t.Fatal("config file is empty")
	}
	if contains(string(data), "s3cr3t_p@ss") {
		t.Error("password should NOT be written to the config file")
	}

	// Reloaded connection should have empty password
	cfg2 := reloadConfig(t, cfg)
	connections, errList := cfg2.ListConnections()
	if errList != nil {
		t.Fatalf("ListConnections failed: %v", errList)
	}
	if connections[0].Password != "" {
		t.Errorf("password after reload = %q, want empty", connections[0].Password)
	}
}

func TestConfig_PasswordFieldCanBeLoaded(t *testing.T) {
	// Verifies that the Password JSON tag can deserialize passwords.
	// If someone changes the tag to json:"-", this test fails because
	// the field can no longer be read from JSON — even though save()
	// intentionally strips it before writing.
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	// Write a config file with a password directly in the JSON
	raw := `{
		"connections": [{
			"id": 1,
			"name": "manual",
			"host": "localhost",
			"port": 6379,
			"password": "loaded_from_json",
			"db": 0,
			"created_at": "2025-01-01T00:00:00Z",
			"updated_at": "2025-01-01T00:00:00Z"
		}]
	}`
	err := os.WriteFile(path, []byte(raw), 0o600)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := NewConfig(path)
	if err != nil {
		t.Fatalf("NewConfig failed: %v", err)
	}

	connections, err := cfg.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections failed: %v", err)
	}
	if len(connections) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(connections))
	}

	// The password field must be readable from JSON — save() strips it,
	// but the struct tag must still support deserialization
	if connections[0].Password != "loaded_from_json" {
		t.Errorf("Password = %q, want %q — the json tag may have been changed to json:\"-\"",
			connections[0].Password, "loaded_from_json")
	}
}

func TestConfig_SSHPasswordFieldCanBeLoaded(t *testing.T) {
	// Same principle: SSH password and passphrase must be loadable from JSON
	// even though save() strips them before writing.
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	raw := `{
		"connections": [{
			"id": 1,
			"name": "ssh-test",
			"host": "localhost",
			"port": 6379,
			"db": 0,
			"use_ssh": true,
			"ssh_config": {
				"host": "bastion",
				"port": 22,
				"user": "deploy",
				"password": "ssh_pass_from_json",
				"passphrase": "key_pass_from_json"
			},
			"created_at": "2025-01-01T00:00:00Z",
			"updated_at": "2025-01-01T00:00:00Z"
		}]
	}`
	err := os.WriteFile(path, []byte(raw), 0o600)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := NewConfig(path)
	if err != nil {
		t.Fatalf("NewConfig failed: %v", err)
	}

	connections, err := cfg.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections failed: %v", err)
	}
	if connections[0].SSHConfig == nil {
		t.Fatal("SSHConfig should not be nil")
	}
	if connections[0].SSHConfig.Password != "ssh_pass_from_json" {
		t.Errorf("SSHConfig.Password = %q, want %q — the json tag may have been changed to json:\"-\"",
			connections[0].SSHConfig.Password, "ssh_pass_from_json")
	}
	if connections[0].SSHConfig.Passphrase != "key_pass_from_json" {
		t.Errorf("SSHConfig.Passphrase = %q, want %q — the json tag may have been changed to json:\"-\"",
			connections[0].SSHConfig.Passphrase, "key_pass_from_json")
	}
}

func TestConfig_Persistence_SSHPasswordStripping(t *testing.T) {
	cfg := newTestConfig(t)

	conn, err := cfg.AddConnection(types.Connection{Name: "ssh-conn", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	// Manually set SSH config with sensitive fields
	cfg.mu.Lock()
	for i := range cfg.Connections {
		if cfg.Connections[i].ID == conn.ID {
			cfg.Connections[i].UseSSH = true
			cfg.Connections[i].SSHConfig = &types.SSHConfig{
				Host:           "bastion.example.com",
				Port:           22,
				User:           "deploy",
				Password:       "ssh_s3cr3t",
				PrivateKeyPath: "/home/user/.ssh/id_rsa",
				Passphrase:     "k3y_p@ss",
			}
		}
	}
	cfg.mu.Unlock()
	if err := cfg.save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Raw JSON should NOT contain SSH password or passphrase
	data, err := os.ReadFile(cfg.path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if contains(string(data), "ssh_s3cr3t") {
		t.Error("SSH password should NOT be written to config file")
	}
	if contains(string(data), "k3y_p@ss") {
		t.Error("SSH passphrase should NOT be written to config file")
	}

	// Non-sensitive SSH fields should survive reload
	cfg2 := reloadConfig(t, cfg)
	connections, errList := cfg2.ListConnections()
	if errList != nil {
		t.Fatalf("ListConnections failed: %v", errList)
	}
	got := connections[0]
	if !got.UseSSH {
		t.Error("UseSSH should be true after reload")
	}
	if got.SSHConfig == nil {
		t.Fatal("SSHConfig should not be nil after reload")
	}
	if got.SSHConfig.Host != "bastion.example.com" {
		t.Errorf("SSHConfig.Host = %q, want %q", got.SSHConfig.Host, "bastion.example.com")
	}
	if got.SSHConfig.Port != 22 {
		t.Errorf("SSHConfig.Port = %d, want %d", got.SSHConfig.Port, 22)
	}
	if got.SSHConfig.User != "deploy" {
		t.Errorf("SSHConfig.User = %q, want %q", got.SSHConfig.User, "deploy")
	}
	if got.SSHConfig.PrivateKeyPath != "/home/user/.ssh/id_rsa" {
		t.Errorf("SSHConfig.PrivateKeyPath = %q, want %q", got.SSHConfig.PrivateKeyPath, "/home/user/.ssh/id_rsa")
	}
	if got.SSHConfig.Password != "" {
		t.Errorf("SSHConfig.Password should be empty after reload, got %q", got.SSHConfig.Password)
	}
	if got.SSHConfig.Passphrase != "" {
		t.Errorf("SSHConfig.Passphrase should be empty after reload, got %q", got.SSHConfig.Passphrase)
	}
}

func TestConfig_Persistence_TLSConfig(t *testing.T) {
	cfg := newTestConfig(t)

	conn, err := cfg.AddConnection(types.Connection{Name: "tls-conn", Host: "localhost", Port: 6380, DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	// Manually set TLS config
	cfg.mu.Lock()
	for i := range cfg.Connections {
		if cfg.Connections[i].ID == conn.ID {
			cfg.Connections[i].UseTLS = true
			cfg.Connections[i].TLSConfig = &types.TLSConfig{
				CertFile:           "/certs/client.pem",
				KeyFile:            "/certs/client-key.pem",
				CAFile:             "/certs/ca.pem",
				InsecureSkipVerify: true,
				ServerName:         "redis.internal",
			}
		}
	}
	cfg.mu.Unlock()
	if err := cfg.save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	cfg2 := reloadConfig(t, cfg)
	connections, err := cfg2.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections failed: %v", err)
	}
	got := connections[0]

	if !got.UseTLS {
		t.Error("UseTLS should be true after reload")
	}
	if got.TLSConfig == nil {
		t.Fatal("TLSConfig should not be nil after reload")
	}
	if got.TLSConfig.CertFile != "/certs/client.pem" {
		t.Errorf("TLSConfig.CertFile = %q, want %q", got.TLSConfig.CertFile, "/certs/client.pem")
	}
	if got.TLSConfig.KeyFile != "/certs/client-key.pem" {
		t.Errorf("TLSConfig.KeyFile = %q, want %q", got.TLSConfig.KeyFile, "/certs/client-key.pem")
	}
	if got.TLSConfig.CAFile != "/certs/ca.pem" {
		t.Errorf("TLSConfig.CAFile = %q, want %q", got.TLSConfig.CAFile, "/certs/ca.pem")
	}
	if !got.TLSConfig.InsecureSkipVerify {
		t.Error("TLSConfig.InsecureSkipVerify should be true after reload")
	}
	if got.TLSConfig.ServerName != "redis.internal" {
		t.Errorf("TLSConfig.ServerName = %q, want %q", got.TLSConfig.ServerName, "redis.internal")
	}
}

// --- Phase 2: UpdateConnection Field Preservation ---

func TestConfig_UpdateConnection_PreservesGroupAndColor(t *testing.T) {
	cfg := newTestConfig(t)

	conn, err := cfg.AddConnection(types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	// Set Group and Color directly (not exposed via AddConnection API)
	cfg.mu.Lock()
	for i := range cfg.Connections {
		if cfg.Connections[i].ID == conn.ID {
			cfg.Connections[i].Group = "Production"
			cfg.Connections[i].Color = "#ff0000"
		}
	}
	cfg.mu.Unlock()

	// Update basic fields
	conn.Name = "renamed"
	conn.Host = "newhost"
	conn.Port = 6380
	conn.Password = "pass"
	conn.DB = 1
	conn.UseCluster = false
	updated, err := cfg.UpdateConnection(conn)
	if err != nil {
		t.Fatalf("UpdateConnection failed: %v", err)
	}

	if updated.Group != "Production" {
		t.Errorf("Group = %q, want %q", updated.Group, "Production")
	}
	if updated.Color != "#ff0000" {
		t.Errorf("Color = %q, want %q", updated.Color, "#ff0000")
	}
}

func TestConfig_UpdateConnection_PreservesSSH(t *testing.T) {
	cfg := newTestConfig(t)

	conn, err := cfg.AddConnection(types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	sshCfg := &types.SSHConfig{
		Host:           "bastion.example.com",
		Port:           22,
		User:           "deploy",
		PrivateKeyPath: "/home/user/.ssh/id_rsa",
	}
	cfg.mu.Lock()
	for i := range cfg.Connections {
		if cfg.Connections[i].ID == conn.ID {
			cfg.Connections[i].UseSSH = true
			cfg.Connections[i].SSHConfig = sshCfg
		}
	}
	cfg.mu.Unlock()

	conn.Name = "renamed"
	conn.Host = "newhost"
	conn.Port = 6380
	conn.DB = 0
	updated, err := cfg.UpdateConnection(conn)
	if err != nil {
		t.Fatalf("UpdateConnection failed: %v", err)
	}

	if !updated.UseSSH {
		t.Error("UseSSH should be preserved after update")
	}
	if updated.SSHConfig == nil {
		t.Fatal("SSHConfig should be preserved after update")
	}
	if updated.SSHConfig.Host != "bastion.example.com" {
		t.Errorf("SSHConfig.Host = %q, want %q", updated.SSHConfig.Host, "bastion.example.com")
	}
	if updated.SSHConfig.User != "deploy" {
		t.Errorf("SSHConfig.User = %q, want %q", updated.SSHConfig.User, "deploy")
	}
}

func TestConfig_UpdateConnection_PreservesTLS(t *testing.T) {
	cfg := newTestConfig(t)

	conn, err := cfg.AddConnection(types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}

	tlsCfg := &types.TLSConfig{
		CertFile:           "/certs/client.pem",
		KeyFile:            "/certs/client-key.pem",
		CAFile:             "/certs/ca.pem",
		InsecureSkipVerify: true,
		ServerName:         "redis.internal",
	}
	cfg.mu.Lock()
	for i := range cfg.Connections {
		if cfg.Connections[i].ID == conn.ID {
			cfg.Connections[i].UseTLS = true
			cfg.Connections[i].TLSConfig = tlsCfg
		}
	}
	cfg.mu.Unlock()

	conn.Name = "renamed"
	conn.Host = "newhost"
	conn.Port = 6380
	conn.DB = 0
	updated, err := cfg.UpdateConnection(conn)
	if err != nil {
		t.Fatalf("UpdateConnection failed: %v", err)
	}

	if !updated.UseTLS {
		t.Error("UseTLS should be preserved after update")
	}
	if updated.TLSConfig == nil {
		t.Fatal("TLSConfig should be preserved after update")
	}
	if updated.TLSConfig.CertFile != "/certs/client.pem" {
		t.Errorf("TLSConfig.CertFile = %q, want %q", updated.TLSConfig.CertFile, "/certs/client.pem")
	}
	if updated.TLSConfig.ServerName != "redis.internal" {
		t.Errorf("TLSConfig.ServerName = %q, want %q", updated.TLSConfig.ServerName, "redis.internal")
	}
}

// --- Phase 3: Remaining Config Persistence ---

func TestConfig_Persistence_Favorites(t *testing.T) {
	cfg := newTestConfig(t)

	_, err := cfg.AddConnection(types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}
	fav, err := cfg.AddFavorite(1, "user:123", "My User Key")
	if err != nil {
		t.Fatalf("AddFavorite failed: %v", err)
	}

	cfg2 := reloadConfig(t, cfg)
	favs := cfg2.ListFavorites(1)
	if len(favs) != 1 {
		t.Fatalf("expected 1 favorite after reload, got %d", len(favs))
	}

	got := favs[0]
	if got.ConnectionID != 1 {
		t.Errorf("ConnectionID = %d, want %d", got.ConnectionID, 1)
	}
	if got.Key != "user:123" {
		t.Errorf("Key = %q, want %q", got.Key, "user:123")
	}
	if got.Label != "My User Key" {
		t.Errorf("Label = %q, want %q", got.Label, "My User Key")
	}
	if got.AddedAt.IsZero() {
		t.Error("AddedAt should not be zero after reload")
	}
	if got.AddedAt.Sub(fav.AddedAt).Abs() > time.Second {
		t.Errorf("AddedAt drifted: got %v, want ~%v", got.AddedAt, fav.AddedAt)
	}
}

func TestConfig_Persistence_RecentKeys(t *testing.T) {
	cfg := newTestConfig(t)

	cfg.AddRecentKey(1, "key1", types.KeyTypeString)
	cfg.AddRecentKey(1, "key2", types.KeyTypeHash)
	cfg.AddRecentKey(2, "key3", types.KeyTypeList)

	cfg2 := reloadConfig(t, cfg)

	// Check connID=1 keys
	recents1 := cfg2.ListRecentKeys(1)
	if len(recents1) != 2 {
		t.Fatalf("expected 2 recent keys for connID=1 after reload, got %d", len(recents1))
	}
	// Most recent first
	if recents1[0].Key != "key2" {
		t.Errorf("recents1[0].Key = %q, want %q", recents1[0].Key, "key2")
	}
	if recents1[0].Type != types.KeyTypeHash {
		t.Errorf("recents1[0].Type = %q, want %q", recents1[0].Type, types.KeyTypeHash)
	}
	if recents1[1].Key != "key1" {
		t.Errorf("recents1[1].Key = %q, want %q", recents1[1].Key, "key1")
	}
	if recents1[1].Type != types.KeyTypeString {
		t.Errorf("recents1[1].Type = %q, want %q", recents1[1].Type, types.KeyTypeString)
	}

	// Check connID=2 isolation
	recents2 := cfg2.ListRecentKeys(2)
	if len(recents2) != 1 {
		t.Fatalf("expected 1 recent key for connID=2 after reload, got %d", len(recents2))
	}
	if recents2[0].Type != types.KeyTypeList {
		t.Errorf("recents2[0].Type = %q, want %q", recents2[0].Type, types.KeyTypeList)
	}
}

func TestConfig_Persistence_Groups(t *testing.T) {
	cfg := newTestConfig(t)

	err := cfg.AddGroup("Production", "#ff0000")
	if err != nil {
		t.Fatalf("AddGroup failed: %v", err)
	}
	conn, err := cfg.AddConnection(types.Connection{Name: "test", Host: "localhost", Port: 6379, DB: 0, UseCluster: false})
	if err != nil {
		t.Fatalf("AddConnection failed: %v", err)
	}
	err = cfg.AddConnectionToGroup("Production", conn.ID)
	if err != nil {
		t.Fatalf("AddConnectionToGroup failed: %v", err)
	}

	cfg2 := reloadConfig(t, cfg)
	groups := cfg2.ListGroups()
	if len(groups) != 1 {
		t.Fatalf("expected 1 group after reload, got %d", len(groups))
	}

	got := groups[0]
	if got.Name != "Production" {
		t.Errorf("Name = %q, want %q", got.Name, "Production")
	}
	if got.Color != "#ff0000" {
		t.Errorf("Color = %q, want %q", got.Color, "#ff0000")
	}
	if len(got.Connections) != 1 {
		t.Fatalf("expected 1 connection in group, got %d", len(got.Connections))
	}
	if got.Connections[0] != conn.ID {
		t.Errorf("Connections[0] = %d, want %d", got.Connections[0], conn.ID)
	}
}

func TestConfig_Persistence_Templates(t *testing.T) {
	cfg := newTestConfig(t)

	custom := types.KeyTemplate{
		Name:         "Custom",
		Description:  "A custom template",
		KeyPattern:   "custom:{id}",
		Type:         types.KeyTypeHash,
		DefaultTTL:   5 * time.Minute,
		DefaultValue: "default",
		Fields:       map[string]string{"field1": "val1", "field2": "val2"},
	}
	err := cfg.AddTemplate(custom)
	if err != nil {
		t.Fatalf("AddTemplate failed: %v", err)
	}

	cfg2 := reloadConfig(t, cfg)
	templates := cfg2.ListTemplates()

	var got *types.KeyTemplate
	for _, tmpl := range templates {
		if tmpl.Name == "Custom" {
			got = &tmpl
			break
		}
	}
	if got == nil {
		t.Fatal("custom template not found after reload")
	}
	if got.Description != "A custom template" {
		t.Errorf("Description = %q, want %q", got.Description, "A custom template")
	}
	if got.KeyPattern != "custom:{id}" {
		t.Errorf("KeyPattern = %q, want %q", got.KeyPattern, "custom:{id}")
	}
	if got.Type != types.KeyTypeHash {
		t.Errorf("Type = %q, want %q", got.Type, types.KeyTypeHash)
	}
	if got.DefaultValue != "default" {
		t.Errorf("DefaultValue = %q, want %q", got.DefaultValue, "default")
	}
	if len(got.Fields) != 2 {
		t.Errorf("Fields count = %d, want 2", len(got.Fields))
	}
	if got.Fields["field1"] != "val1" {
		t.Errorf("Fields[field1] = %q, want %q", got.Fields["field1"], "val1")
	}
}

func TestConfig_Persistence_KeyBindings(t *testing.T) {
	cfg := newTestConfig(t)

	bindings := cfg.GetKeyBindings()
	bindings.Quit = "ctrl+x"
	err := cfg.SetKeyBindings(bindings)
	if err != nil {
		t.Fatalf("SetKeyBindings failed: %v", err)
	}

	cfg2 := reloadConfig(t, cfg)
	got := cfg2.GetKeyBindings()
	if got.Quit != "ctrl+x" {
		t.Errorf("Quit = %q, want %q after reload", got.Quit, "ctrl+x")
	}
}

func TestConfig_Persistence_TreeSeparator(t *testing.T) {
	cfg := newTestConfig(t)

	err := cfg.SetTreeSeparator("/")
	if err != nil {
		t.Fatalf("SetTreeSeparator failed: %v", err)
	}

	cfg2 := reloadConfig(t, cfg)
	if cfg2.GetTreeSeparator() != "/" {
		t.Errorf("TreeSeparator = %q, want %q after reload", cfg2.GetTreeSeparator(), "/")
	}
}

func TestConfig_Persistence_ValueHistory_NotPersisted(t *testing.T) {
	// ValueHistory uses json:"-" so Redis values (which may contain secrets)
	// are never written to disk. History is in-memory only.
	cfg := newTestConfig(t)

	value := types.RedisValue{
		Type:        types.KeyTypeString,
		StringValue: "hello world",
	}
	cfg.AddValueHistory("user:123", value, "set")

	// Verify it exists in memory.
	if len(cfg.GetValueHistory("user:123")) != 1 {
		t.Fatal("expected 1 history entry in memory")
	}

	// After reload, history must be gone.
	cfg2 := reloadConfig(t, cfg)
	history := cfg2.GetValueHistory("user:123")
	if len(history) != 0 {
		t.Errorf("expected 0 history entries after reload (json:\"-\"), got %d", len(history))
	}
}

func TestConfig_Persistence_Settings(t *testing.T) {
	cfg := newTestConfig(t)

	cfg.mu.Lock()
	cfg.MaxRecentKeys = 50
	cfg.MaxValueHistory = 100
	cfg.WatchInterval = 2000
	cfg.mu.Unlock()
	if err := cfg.save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	cfg2 := reloadConfig(t, cfg)
	if cfg2.MaxRecentKeys != 50 {
		t.Errorf("MaxRecentKeys = %d, want 50 after reload", cfg2.MaxRecentKeys)
	}
	if cfg2.MaxValueHistory != 100 {
		t.Errorf("MaxValueHistory = %d, want 100 after reload", cfg2.MaxValueHistory)
	}
	if cfg2.WatchInterval != 2000 {
		t.Errorf("WatchInterval = %d, want 2000 after reload", cfg2.WatchInterval)
	}
}

// --- Phase 4: Untested Methods ---

func TestConfig_ClearRecentKeys(t *testing.T) {
	cfg := newTestConfig(t)

	cfg.AddRecentKey(1, "key1", types.KeyTypeString)
	cfg.AddRecentKey(1, "key2", types.KeyTypeHash)
	cfg.AddRecentKey(2, "key3", types.KeyTypeList)

	cfg.ClearRecentKeys(1)

	// connID=1 should be empty
	if len(cfg.ListRecentKeys(1)) != 0 {
		t.Errorf("expected 0 recent keys for connID=1 after clear, got %d", len(cfg.ListRecentKeys(1)))
	}

	// connID=2 should be untouched
	if len(cfg.ListRecentKeys(2)) != 1 {
		t.Errorf("expected 1 recent key for connID=2 after clear, got %d", len(cfg.ListRecentKeys(2)))
	}
}

func TestConfig_ClearRecentKeys_Persistence(t *testing.T) {
	cfg := newTestConfig(t)

	cfg.AddRecentKey(1, "key1", types.KeyTypeString)
	cfg.ClearRecentKeys(1)

	cfg2 := reloadConfig(t, cfg)
	if len(cfg2.ListRecentKeys(1)) != 0 {
		t.Errorf("expected 0 recent keys after reload, got %d", len(cfg2.ListRecentKeys(1)))
	}
}

func TestConfig_ClearValueHistory(t *testing.T) {
	cfg := newTestConfig(t)

	value := types.RedisValue{Type: types.KeyTypeString, StringValue: "test"}
	cfg.AddValueHistory("key1", value, "set")
	cfg.AddValueHistory("key2", value, "set")

	cfg.ClearValueHistory()

	if len(cfg.GetValueHistory("key1")) != 0 {
		t.Error("expected empty history for key1 after clear")
	}
	if len(cfg.GetValueHistory("key2")) != 0 {
		t.Error("expected empty history for key2 after clear")
	}
}

func TestConfig_ClearValueHistory_Persistence(t *testing.T) {
	cfg := newTestConfig(t)

	value := types.RedisValue{Type: types.KeyTypeString, StringValue: "test"}
	cfg.AddValueHistory("key1", value, "set")
	cfg.ClearValueHistory()

	cfg2 := reloadConfig(t, cfg)
	if len(cfg2.GetValueHistory("key1")) != 0 {
		t.Errorf("expected empty history after reload, got %d entries", len(cfg2.GetValueHistory("key1")))
	}
}
