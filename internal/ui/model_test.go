package ui

import (
	"testing"

	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestNewModel(t *testing.T) {
	m := NewModel()

	// Check initial screen
	if m.Screen != types.ScreenConnections {
		t.Errorf("Screen = %v, want %v", m.Screen, types.ScreenConnections)
	}

	// Check inputs are initialized
	if len(m.ConnInputs) != 9 {
		t.Errorf("ConnInputs length = %d, want 9", len(m.ConnInputs))
	}

	if len(m.AddKeyInputs) != 3 {
		t.Errorf("AddKeyInputs length = %d, want 3", len(m.AddKeyInputs))
	}

	if len(m.AddCollectionInput) != 2 {
		t.Errorf("AddCollectionInput length = %d, want 2", len(m.AddCollectionInput))
	}

	if len(m.PubSubInput) != 2 {
		t.Errorf("PubSubInput length = %d, want 2", len(m.PubSubInput))
	}

	// Check default values in conn inputs
	if m.ConnInputs[1].Value() != "localhost" {
		t.Errorf("Host default = %q, want \"localhost\"", m.ConnInputs[1].Value())
	}

	if m.ConnInputs[2].Value() != "6379" {
		t.Errorf("Port default = %q, want \"6379\"", m.ConnInputs[2].Value())
	}

	if m.ConnInputs[8].Value() != "0" {
		t.Errorf("DB default = %q, want \"0\"", m.ConnInputs[8].Value())
	}

	// Check TreeExpanded map is initialized
	if m.TreeExpanded == nil {
		t.Error("TreeExpanded should be initialized")
	}

	// Check SelectedBulkKeys map is initialized
	if m.SelectedBulkKeys == nil {
		t.Error("SelectedBulkKeys should be initialized")
	}

	// Check default tree separator
	if m.TreeSeparator != ":" {
		t.Errorf("TreeSeparator = %q, want \":\"", m.TreeSeparator)
	}

	// Check default add key type
	if m.AddKeyType != types.KeyTypeString {
		t.Errorf("AddKeyType = %v, want %v", m.AddKeyType, types.KeyTypeString)
	}
}

func TestModel_GetPort(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected int
	}{
		{"valid port", "6379", 6379},
		{"custom port", "6380", 6380},
		{"empty returns default", "", 6379},
		{"invalid returns default", "invalid", 6379},
		{"negative returns parsed value", "-1", -1}, // strconv.Atoi accepts negatives
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel()
			m.ConnInputs[2].SetValue(tt.value)

			got := m.getPort()
			if got != tt.expected {
				t.Errorf("getPort() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestModel_GetDB(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected int
	}{
		{"db 0", "0", 0},
		{"db 1", "1", 1},
		{"db 15", "15", 15},
		{"empty returns default", "", 0},
		{"invalid returns default", "invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel()
			m.ConnInputs[8].SetValue(tt.value)

			got := m.getDB()
			if got != tt.expected {
				t.Errorf("getDB() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestModel_ResetConnInputs(t *testing.T) {
	m := NewModel()

	// Set some values
	m.ConnInputs[0].SetValue("My Connection")
	m.ConnInputs[1].SetValue("redis.example.com")
	m.ConnInputs[2].SetValue("6380")
	m.ConnInputs[3].SetValue("user")
	m.ConnInputs[4].SetValue("secret")
	m.ConnInputs[5].SetValue("secret/data/redis")
	m.ConnInputs[6].SetValue("credentials.redis.username")
	m.ConnInputs[7].SetValue("credentials.redis.password")
	m.ConnInputs[8].SetValue("5")
	m.ConnFocusIdx = 3
	m.ConnUseVault = true

	// Reset
	m.resetConnInputs()

	// Check values are reset to defaults
	if m.ConnInputs[0].Value() != "" {
		t.Errorf("Name should be empty, got %q", m.ConnInputs[0].Value())
	}
	if m.ConnInputs[1].Value() != "localhost" {
		t.Errorf("Host = %q, want \"localhost\"", m.ConnInputs[1].Value())
	}
	if m.ConnInputs[2].Value() != "6379" {
		t.Errorf("Port = %q, want \"6379\"", m.ConnInputs[2].Value())
	}
	if m.ConnInputs[3].Value() != "" {
		t.Errorf("Username should be empty, got %q", m.ConnInputs[3].Value())
	}
	if m.ConnInputs[4].Value() != "" {
		t.Errorf("Password should be empty, got %q", m.ConnInputs[4].Value())
	}
	if m.ConnInputs[5].Value() != "" || m.ConnInputs[6].Value() != "" || m.ConnInputs[7].Value() != "" {
		t.Errorf("Vault reference should be empty, got %q/%q/%q", m.ConnInputs[5].Value(), m.ConnInputs[6].Value(), m.ConnInputs[7].Value())
	}
	if m.ConnInputs[8].Value() != "0" {
		t.Errorf("DB = %q, want \"0\"", m.ConnInputs[8].Value())
	}
	if m.ConnFocusIdx != 0 {
		t.Errorf("ConnFocusIdx = %d, want 0", m.ConnFocusIdx)
	}
	if m.ConnUseVault {
		t.Error("ConnUseVault should be reset to false")
	}
}

func TestModel_ResetAddKeyInputs(t *testing.T) {
	m := NewModel()

	// Set some values
	m.AddKeyInputs[0].SetValue("user:123")
	m.AddKeyInputs[1].SetValue("some value")
	m.AddKeyInputs[2].SetValue("extra")
	m.AddKeyFocusIdx = 2
	m.AddKeyType = types.KeyTypeHash

	// Reset
	m.resetAddKeyInputs()

	// Check values are reset
	if m.AddKeyInputs[0].Value() != "" {
		t.Errorf("Key name should be empty, got %q", m.AddKeyInputs[0].Value())
	}
	if m.AddKeyInputs[1].Value() != "" {
		t.Errorf("Value should be empty, got %q", m.AddKeyInputs[1].Value())
	}
	if m.AddKeyInputs[2].Value() != "" {
		t.Errorf("Extra should be empty, got %q", m.AddKeyInputs[2].Value())
	}
	if m.AddKeyFocusIdx != 0 {
		t.Errorf("AddKeyFocusIdx = %d, want 0", m.AddKeyFocusIdx)
	}
	if m.AddKeyType != types.KeyTypeString {
		t.Errorf("AddKeyType = %v, want %v", m.AddKeyType, types.KeyTypeString)
	}
}

func TestModel_PopulateConnInputs(t *testing.T) {
	m := NewModel()

	conn := types.Connection{
		Name:             "Production",
		Host:             "redis.prod.com",
		Port:             6380,
		Username:         "default",
		Password:         "supersecret",
		VaultPath:        "secret/data/redis",
		VaultUserKey:     "credentials.redis.username",
		VaultPasswordKey: "credentials.redis.password",
		DB:               2,
	}

	m.populateConnInputs(conn)

	if m.ConnInputs[0].Value() != "Production" {
		t.Errorf("Name = %q, want \"Production\"", m.ConnInputs[0].Value())
	}
	if m.ConnInputs[1].Value() != "redis.prod.com" {
		t.Errorf("Host = %q, want \"redis.prod.com\"", m.ConnInputs[1].Value())
	}
	if m.ConnInputs[2].Value() != "6380" {
		t.Errorf("Port = %q, want \"6380\"", m.ConnInputs[2].Value())
	}
	if m.ConnInputs[3].Value() != "default" {
		t.Errorf("Username = %q, want \"default\"", m.ConnInputs[3].Value())
	}
	if m.ConnInputs[4].Value() != "supersecret" {
		t.Errorf("Password = %q, want \"supersecret\"", m.ConnInputs[4].Value())
	}
	if m.ConnInputs[5].Value() != "secret/data/redis" || m.ConnInputs[6].Value() != "credentials.redis.username" || m.ConnInputs[7].Value() != "credentials.redis.password" {
		t.Errorf("Vault reference = %q/%q/%q", m.ConnInputs[5].Value(), m.ConnInputs[6].Value(), m.ConnInputs[7].Value())
	}
	if m.ConnInputs[8].Value() != "2" {
		t.Errorf("DB = %q, want \"2\"", m.ConnInputs[8].Value())
	}
	if !m.ConnUseVault {
		t.Error("ConnUseVault should be on when the connection has vault fields")
	}

	m.populateConnInputs(types.Connection{Name: "plain", Host: "localhost", Port: 6379})
	if m.ConnUseVault {
		t.Error("ConnUseVault should be off when the connection has no vault fields")
	}
}

func TestModel_ConvertCurrentInputsToConnection_Add(t *testing.T) {
	m := NewModel()

	// Set some values / m state
	m.ConnInputs[0].SetValue("My Connection")
	m.ConnInputs[1].SetValue("redis.example.com")
	m.ConnInputs[2].SetValue("6380")
	m.ConnInputs[3].SetValue("user")
	m.ConnInputs[4].SetValue("secret")
	m.ConnInputs[5].SetValue("secret/data/redis")
	m.ConnInputs[6].SetValue("credentials.redis.username")
	m.ConnInputs[7].SetValue("credentials.redis.password")
	m.ConnInputs[8].SetValue("5")
	m.ConnClusterMode = true
	m.ConnUseVault = true
	m.ConnFocusIdx = 3

	// Convert
	conn := m.convertCurrentInputsToConnection(m.ConnInputs, "add")

	if conn.Name != "My Connection" {
		t.Errorf("Name = %q, want %q", conn.Name, "My Connection")
	}
	if conn.Host != "redis.example.com" {
		t.Errorf("Host = %q, want %q", conn.Host, "redis.example.com")
	}
	if conn.Port != 6380 {
		t.Errorf("Port = %d, want %d", conn.Port, 6380)
	}
	if conn.Username != "user" {
		t.Errorf("Username = %q, want %q", conn.Username, "user")
	}
	if conn.Password != "secret" {
		t.Errorf("Password = %q, want %q", conn.Password, "secret")
	}
	if conn.VaultPath != "secret/data/redis" || conn.VaultUserKey != "credentials.redis.username" || conn.VaultPasswordKey != "credentials.redis.password" {
		t.Errorf("Vault reference = %q/%q/%q", conn.VaultPath, conn.VaultUserKey, conn.VaultPasswordKey)
	}
	if conn.DB != 5 {
		t.Errorf("DB = %d, want %d", conn.DB, 5)
	}
	if conn.UseCluster != true {
		t.Errorf("UseCluster = %v, want %v", conn.UseCluster, true)
	}
}

func TestModel_ConvertCurrentInputsToConnection_Edit(t *testing.T) {
	m := NewModel()

	// Set some values / m state
	m.EditingConnection = &types.Connection{ID: 1, Name: "Old", Host: "localhost", Port: 6379, DB: 0, UseCluster: false}
	m.ConnInputs[0].SetValue("My Connection")
	m.ConnInputs[1].SetValue("redis.example.com")
	m.ConnInputs[2].SetValue("6380")
	m.ConnInputs[3].SetValue("user")
	m.ConnInputs[4].SetValue("secret")
	m.ConnInputs[5].SetValue("secret/data/redis")
	m.ConnInputs[6].SetValue("credentials.redis.username")
	m.ConnInputs[7].SetValue("credentials.redis.password")
	m.ConnInputs[8].SetValue("5")
	m.ConnClusterMode = true
	m.ConnUseVault = true
	m.ConnFocusIdx = 3

	// Convert
	conn := m.convertCurrentInputsToConnection(m.ConnInputs, "edit")

	if conn.Name != "My Connection" {
		t.Errorf("Name = %q, want %q", conn.Name, "My Connection")
	}
	if conn.Host != "redis.example.com" {
		t.Errorf("Host = %q, want %q", conn.Host, "redis.example.com")
	}
	if conn.Port != 6380 {
		t.Errorf("Port = %d, want %d", conn.Port, 6380)
	}
	if conn.Username != "user" {
		t.Errorf("Username = %q, want %q", conn.Username, "user")
	}
	if conn.Password != "secret" {
		t.Errorf("Password = %q, want %q", conn.Password, "secret")
	}
	if conn.VaultPath != "secret/data/redis" || conn.VaultUserKey != "credentials.redis.username" || conn.VaultPasswordKey != "credentials.redis.password" {
		t.Errorf("Vault reference = %q/%q/%q", conn.VaultPath, conn.VaultUserKey, conn.VaultPasswordKey)
	}
	if conn.DB != 5 {
		t.Errorf("DB = %d, want %d", conn.DB, 5)
	}
	if conn.UseCluster != true {
		t.Errorf("UseCluster = %v, want %v", conn.UseCluster, true)
	}
	if conn.ID != 1 {
		t.Errorf("ID = %d, want %d", conn.ID, 1)
	}
}

func TestModel_ConvertCurrentInputsToConnection_VaultDisabled(t *testing.T) {
	m := NewModel()

	m.ConnInputs[0].SetValue("My Connection")
	m.ConnInputs[1].SetValue("redis.example.com")
	m.ConnInputs[5].SetValue("secret/data/redis")
	m.ConnInputs[6].SetValue("credentials.redis.username")
	m.ConnInputs[7].SetValue("credentials.redis.password")
	m.ConnUseVault = false

	conn := m.convertCurrentInputsToConnection(m.ConnInputs, "add")

	if conn.VaultPath != "" || conn.VaultUserKey != "" || conn.VaultPasswordKey != "" {
		t.Errorf("Vault reference should be blank when vault is off, got %q/%q/%q", conn.VaultPath, conn.VaultUserKey, conn.VaultPasswordKey)
	}
}

func TestModel_ResetAddCollectionInputs(t *testing.T) {
	m := NewModel()

	// Set some values
	m.AddCollectionInput[0].SetValue("member1")
	m.AddCollectionInput[1].SetValue("100")
	m.AddCollFocusIdx = 1

	// Reset
	m.resetAddCollectionInputs()

	// Check values are reset
	if m.AddCollectionInput[0].Value() != "" {
		t.Errorf("First input should be empty, got %q", m.AddCollectionInput[0].Value())
	}
	if m.AddCollectionInput[1].Value() != "" {
		t.Errorf("Second input should be empty, got %q", m.AddCollectionInput[1].Value())
	}
	if m.AddCollFocusIdx != 0 {
		t.Errorf("AddCollFocusIdx = %d, want 0", m.AddCollFocusIdx)
	}
}

func TestModel_ResetPubSubInputs(t *testing.T) {
	m := NewModel()

	// Set some values
	m.PubSubInput[0].SetValue("notifications")
	m.PubSubInput[1].SetValue("hello world")
	m.PubSubFocusIdx = 1

	// Reset
	m.resetPubSubInputs()

	// Check values are reset
	if m.PubSubInput[0].Value() != "" {
		t.Errorf("Channel should be empty, got %q", m.PubSubInput[0].Value())
	}
	if m.PubSubInput[1].Value() != "" {
		t.Errorf("Message should be empty, got %q", m.PubSubInput[1].Value())
	}
	if m.PubSubFocusIdx != 0 {
		t.Errorf("PubSubFocusIdx = %d, want 0", m.PubSubFocusIdx)
	}
}

func TestModel_CLIConnection_Nil(t *testing.T) {
	m := NewModel()
	if m.CLIConnection != nil {
		t.Error("CLIConnection should be nil by default")
	}
}

func TestModel_HandleAutoConnectMsg(t *testing.T) {
	m := NewModel()
	conn := types.Connection{
		Name:     "test",
		Host:     "redis.example.com",
		Port:     6380,
		Username: "default",
		Password: "password",
		DB:       2,
	}
	msg := types.AutoConnectMsg{Connection: conn}

	result, _ := m.handleAutoConnectMsg(msg)
	model := result.(Model)

	if model.CurrentConn == nil {
		t.Fatal("CurrentConn should be set")
	}
	if model.CurrentConn.Host != "redis.example.com" {
		t.Errorf("Host = %q, want %q", model.CurrentConn.Host, "redis.example.com")
	}
	if model.CurrentConn.Port != 6380 {
		t.Errorf("Port = %d, want %d", model.CurrentConn.Port, 6380)
	}
	if model.CurrentConn.Username != "default" {
		t.Errorf("Username = %q, want %q", model.CurrentConn.Username, "default")
	}
	if model.CurrentConn.Password != "password" {
		t.Errorf("Password = %q, want %q", model.CurrentConn.Password, "password")
	}
	if model.CurrentConn.DB != 2 {
		t.Errorf("DB = %d, want %d", model.CurrentConn.DB, 2)
	}
	if !model.Loading {
		t.Error("Loading should be true")
	}
	if model.StatusMsg != "Connecting..." {
		t.Errorf("StatusMsg = %q, want %q", model.StatusMsg, "Connecting...")
	}
	if model.CLIConnection != nil {
		t.Error("CLIConnection should be nil after handling (consumed)")
	}
}

func TestModel_HandleAutoConnectMsg_Cluster(t *testing.T) {
	m := NewModel()
	conn := types.Connection{
		Name:       "cluster",
		Host:       "redis.example.com",
		Port:       7000,
		UseCluster: true,
	}
	msg := types.AutoConnectMsg{Connection: conn}

	result, _ := m.handleAutoConnectMsg(msg)
	model := result.(Model)

	if model.CurrentConn == nil {
		t.Fatal("CurrentConn should be set")
	}
	if !model.CurrentConn.UseCluster {
		t.Error("UseCluster should be true")
	}
}

// Note: Tests for unexported functions like createConnInputs, createAddKeyInputs,
// createAddCollectionInputs, and createPubSubInputs are covered indirectly
// through TestNewModel which verifies the inputs are correctly initialized.
