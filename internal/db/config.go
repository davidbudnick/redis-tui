package db

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"
)

// jsonMarshalIndent is overridable in tests to simulate marshal errors.
var jsonMarshalIndent = json.MarshalIndent

// Config stores all application configuration
type Config struct {
	Connections     []types.Connection        `json:"connections"`
	Groups          []types.ConnectionGroup   `json:"groups,omitempty"`
	Favorites       []types.Favorite          `json:"favorites,omitempty"`
	RecentKeys      []types.RecentKey         `json:"recent_keys,omitempty"`
	Templates       []types.KeyTemplate       `json:"templates,omitempty"`
	KeyBindings     types.KeyBindings         `json:"key_bindings"`
	TreeSeparator   string                    `json:"tree_separator"`
	ValueHistory    []types.ValueHistoryEntry `json:"-"` // In-memory only — Redis values may contain secrets.
	MaxRecentKeys   int                       `json:"max_recent_keys"`
	MaxValueHistory int                       `json:"max_value_history"`
	WatchInterval   int                       `json:"watch_interval_ms"`
	nextID          int64
	path            string
	mu              sync.RWMutex
}

func NewConfig(configPath string) (*Config, error) {
	c := &Config{
		path:            configPath,
		Connections:     []types.Connection{},
		Groups:          []types.ConnectionGroup{},
		Favorites:       []types.Favorite{},
		RecentKeys:      []types.RecentKey{},
		Templates:       defaultTemplates(),
		KeyBindings:     types.DefaultKeyBindings(),
		TreeSeparator:   ":",
		MaxRecentKeys:   20,
		MaxValueHistory: 50,
		WatchInterval:   1000,
		nextID:          1,
	}

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}

	// Try to load existing config
	if err := c.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// Calculate next ID
	for _, conn := range c.Connections {
		if conn.ID >= c.nextID {
			c.nextID = conn.ID + 1
		}
	}

	return c, nil
}

func defaultTemplates() []types.KeyTemplate {
	return []types.KeyTemplate{
		{
			Name:        "Session",
			Description: "User session data",
			KeyPattern:  "session:{user_id}",
			Type:        types.KeyTypeHash,
			DefaultTTL:  24 * time.Hour,
			Fields:      map[string]string{"token": "", "created_at": "", "user_agent": ""},
		},
		{
			Name:        "Cache",
			Description: "Cached data with TTL",
			KeyPattern:  "cache:{resource}:{id}",
			Type:        types.KeyTypeString,
			DefaultTTL:  1 * time.Hour,
		},
		{
			Name:         "Rate Limit",
			Description:  "Rate limiting counter",
			KeyPattern:   "ratelimit:{ip}:{endpoint}",
			Type:         types.KeyTypeString,
			DefaultTTL:   1 * time.Minute,
			DefaultValue: "0",
		},
		{
			Name:        "Queue",
			Description: "Job queue",
			KeyPattern:  "queue:{name}",
			Type:        types.KeyTypeList,
		},
		{
			Name:        "Leaderboard",
			Description: "Sorted leaderboard",
			KeyPattern:  "leaderboard:{game}",
			Type:        types.KeyTypeZSet,
		},
	}
}

func (c *Config) load() error {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, c)
}

func (c *Config) save() error {
	// Create a copy of connections without passwords for JSON serialization
	safeConnections := make([]types.Connection, len(c.Connections))
	for i, conn := range c.Connections {
		safeConnections[i] = conn
		safeConnections[i].Password = "" // Don't save password to JSON
		if safeConnections[i].SSHConfig != nil {
			sshCopy := *conn.SSHConfig
			sshCopy.Password = ""
			sshCopy.Passphrase = ""
			safeConnections[i].SSHConfig = &sshCopy
		}
	}

	// Create a copy of config with safe connections
	safeCfg := &Config{
		Connections:     safeConnections,
		Groups:          c.Groups,
		Favorites:       c.Favorites,
		RecentKeys:      c.RecentKeys,
		Templates:       c.Templates,
		KeyBindings:     c.KeyBindings,
		TreeSeparator:   c.TreeSeparator,
		ValueHistory:    c.ValueHistory,
		MaxRecentKeys:   c.MaxRecentKeys,
		MaxValueHistory: c.MaxValueHistory,
		WatchInterval:   c.WatchInterval,
	}

	data, err := jsonMarshalIndent(safeCfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.path, data, 0o600)
}

func (c *Config) Close() error {
	return nil
}

func (c *Config) ListConnections() ([]types.Connection, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]types.Connection, len(c.Connections))
	copy(result, c.Connections)

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	return result, nil
}

func (c *Config) AddConnection(conn types.Connection) (types.Connection, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	conn.ID = c.nextID
	conn.Created = now
	conn.Updated = now

	c.nextID++

	c.Connections = append(c.Connections, conn)

	if err := c.save(); err != nil {
		c.Connections = c.Connections[:len(c.Connections)-1]
		c.nextID--
		return types.Connection{}, err
	}

	addedConnection := c.Connections[len(c.Connections)-1]

	return addedConnection, nil
}

func (c *Config) UpdateConnection(conn types.Connection) (types.Connection, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, toUpdateConn := range c.Connections {
		if toUpdateConn.ID == conn.ID {
			now := time.Now()
			updatedConn := types.Connection{
				ID:               conn.ID,
				Name:             conn.Name,
				Host:             conn.Host,
				Port:             conn.Port,
				Username:         conn.Username,
				Password:         conn.Password,
				VaultPath:        conn.VaultPath,
				VaultUserKey:     conn.VaultUserKey,
				VaultPasswordKey: conn.VaultPasswordKey,
				DB:               conn.DB,
				Group:            toUpdateConn.Group,
				Color:            toUpdateConn.Color,
				UseSSH:           toUpdateConn.UseSSH,
				SSHConfig:        toUpdateConn.SSHConfig,
				UseTLS:           toUpdateConn.UseTLS,
				TLSConfig:        toUpdateConn.TLSConfig,
				UseCluster:       conn.UseCluster,
				Created:          toUpdateConn.Created,
				Updated:          now,
			}

			c.Connections[i] = updatedConn

			if err := c.save(); err != nil {
				c.Connections[i] = toUpdateConn // Rollback
				return types.Connection{}, err
			}

			return updatedConn, nil
		}
	}

	return types.Connection{}, os.ErrNotExist
}

func (c *Config) DeleteConnection(id int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, conn := range c.Connections {
		if conn.ID == id {
			c.Connections = append(c.Connections[:i], c.Connections[i+1:]...)
			return c.save()
		}
	}

	return os.ErrNotExist
}

// Favorites management

func (c *Config) AddFavorite(connID int64, key, label string) (types.Favorite, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if already exists
	for _, f := range c.Favorites {
		if f.ConnectionID == connID && f.Key == key {
			return f, nil
		}
	}

	fav := types.Favorite{
		ConnectionID: connID,
		Key:          key,
		Label:        label,
		AddedAt:      time.Now(),
	}

	c.Favorites = append(c.Favorites, fav)

	if err := c.save(); err != nil {
		c.Favorites = c.Favorites[:len(c.Favorites)-1]
		return types.Favorite{}, err
	}

	return fav, nil
}

func (c *Config) RemoveFavorite(connID int64, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, f := range c.Favorites {
		if f.ConnectionID == connID && f.Key == key {
			c.Favorites = append(c.Favorites[:i], c.Favorites[i+1:]...)
			return c.save()
		}
	}

	return os.ErrNotExist
}

func (c *Config) ListFavorites(connID int64) []types.Favorite {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []types.Favorite
	for _, f := range c.Favorites {
		if f.ConnectionID == connID {
			result = append(result, f)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].AddedAt.After(result[j].AddedAt)
	})

	return result
}

func (c *Config) IsFavorite(connID int64, key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, f := range c.Favorites {
		if f.ConnectionID == connID && f.Key == key {
			return true
		}
	}
	return false
}

// Recent keys management

func (c *Config) AddRecentKey(connID int64, key string, keyType types.KeyType) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove if already exists
	for i, r := range c.RecentKeys {
		if r.ConnectionID == connID && r.Key == key {
			c.RecentKeys = append(c.RecentKeys[:i], c.RecentKeys[i+1:]...)
			break
		}
	}

	recent := types.RecentKey{
		ConnectionID: connID,
		Key:          key,
		Type:         keyType,
		AccessedAt:   time.Now(),
	}

	c.RecentKeys = append([]types.RecentKey{recent}, c.RecentKeys...)

	// Trim to max
	if len(c.RecentKeys) > c.MaxRecentKeys {
		c.RecentKeys = c.RecentKeys[:c.MaxRecentKeys]
	}

	_ = c.save()
}

func (c *Config) ListRecentKeys(connID int64) []types.RecentKey {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []types.RecentKey
	for _, r := range c.RecentKeys {
		if r.ConnectionID == connID {
			result = append(result, r)
		}
	}
	return result
}

func (c *Config) ClearRecentKeys(connID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var remaining []types.RecentKey
	for _, r := range c.RecentKeys {
		if r.ConnectionID != connID {
			remaining = append(remaining, r)
		}
	}
	c.RecentKeys = remaining
	_ = c.save()
}

// Value history management

func (c *Config) AddValueHistory(key string, value types.RedisValue, action string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := types.ValueHistoryEntry{
		Key:       key,
		Value:     value,
		Timestamp: time.Now(),
		Action:    action,
	}

	c.ValueHistory = append([]types.ValueHistoryEntry{entry}, c.ValueHistory...)

	// Trim to max
	if len(c.ValueHistory) > c.MaxValueHistory {
		c.ValueHistory = c.ValueHistory[:c.MaxValueHistory]
	}

	_ = c.save()
}

func (c *Config) GetValueHistory(key string) []types.ValueHistoryEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []types.ValueHistoryEntry
	for _, h := range c.ValueHistory {
		if h.Key == key {
			result = append(result, h)
		}
	}
	return result
}

func (c *Config) ClearValueHistory() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ValueHistory = []types.ValueHistoryEntry{}
	_ = c.save()
}

// Templates management

func (c *Config) ListTemplates() []types.KeyTemplate {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]types.KeyTemplate, len(c.Templates))
	copy(result, c.Templates)
	return result
}

func (c *Config) AddTemplate(t types.KeyTemplate) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Templates = append(c.Templates, t)
	return c.save()
}

func (c *Config) DeleteTemplate(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, t := range c.Templates {
		if t.Name == name {
			c.Templates = append(c.Templates[:i], c.Templates[i+1:]...)
			return c.save()
		}
	}
	return os.ErrNotExist
}

// Connection groups management

func (c *Config) ListGroups() []types.ConnectionGroup {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]types.ConnectionGroup, len(c.Groups))
	copy(result, c.Groups)
	return result
}

func (c *Config) AddGroup(name, color string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	group := types.ConnectionGroup{
		Name:        name,
		Color:       color,
		Connections: []int64{},
	}

	c.Groups = append(c.Groups, group)
	return c.save()
}

func (c *Config) AddConnectionToGroup(groupName string, connID int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, g := range c.Groups {
		if g.Name == groupName {
			// Check if already in group
			for _, id := range g.Connections {
				if id == connID {
					return nil
				}
			}
			c.Groups[i].Connections = append(c.Groups[i].Connections, connID)
			return c.save()
		}
	}
	return os.ErrNotExist
}

func (c *Config) RemoveConnectionFromGroup(groupName string, connID int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, g := range c.Groups {
		if g.Name == groupName {
			for j, id := range g.Connections {
				if id == connID {
					c.Groups[i].Connections = append(g.Connections[:j], g.Connections[j+1:]...)
					return c.save()
				}
			}
		}
	}
	return os.ErrNotExist
}

// KeyBindings management

func (c *Config) GetKeyBindings() types.KeyBindings {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.KeyBindings
}

func (c *Config) SetKeyBindings(kb types.KeyBindings) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.KeyBindings = kb
	return c.save()
}

func (c *Config) ResetKeyBindings() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.KeyBindings = types.DefaultKeyBindings()
	return c.save()
}

// Settings

func (c *Config) GetTreeSeparator() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.TreeSeparator == "" {
		return ":"
	}
	return c.TreeSeparator
}

func (c *Config) SetTreeSeparator(sep string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.TreeSeparator = sep
	return c.save()
}

func (c *Config) GetWatchInterval() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.WatchInterval <= 0 {
		return time.Second
	}
	return time.Duration(c.WatchInterval) * time.Millisecond
}
