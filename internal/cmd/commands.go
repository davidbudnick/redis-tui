package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/davidbudnick/redis-tui/internal/service"
	"github.com/davidbudnick/redis-tui/internal/types"

	tea "github.com/charmbracelet/bubbletea"
)

// Commands provides tea.Cmd factories with injected dependencies.
// Use this struct instead of global functions for better testability.
type Commands struct {
	config service.ConfigService
	redis  service.RedisService
}

// NewCommands creates a new Commands instance with the provided services.
func NewCommands(config service.ConfigService, redis service.RedisService) *Commands {
	return &Commands{
		config: config,
		redis:  redis,
	}
}

// NewCommandsFromContainer creates a new Commands instance from a service container.
func NewCommandsFromContainer(c *service.Container) *Commands {
	return &Commands{
		config: c.Config,
		redis:  c.Redis,
	}
}

// Connection commands

func (c *Commands) LoadConnections() tea.Cmd {
	return func() tea.Msg {
		if c.config == nil {
			return types.ConnectionsLoadedMsg{Err: nil}
		}
		connections, err := c.config.ListConnections()
		if err != nil {
			slog.Error("Failed to load connections", "error", err)
		}
		return types.ConnectionsLoadedMsg{Connections: connections, Err: err}
	}
}

func (c *Commands) AddConnection(name, host string, port int, password string, dbNum int, useCluster bool) tea.Cmd {
	return func() tea.Msg {
		if c.config == nil {
			return types.ConnectionAddedMsg{Err: nil}
		}
		conn, err := c.config.AddConnection(name, host, port, password, dbNum, useCluster)
		if err != nil {
			slog.Error("Failed to add connection", "error", err)
		}
		return types.ConnectionAddedMsg{Connection: conn, Err: err}
	}
}

func (c *Commands) UpdateConnection(id int64, name, host string, port int, password string, dbNum int, useCluster bool) tea.Cmd {
	return func() tea.Msg {
		if c.config == nil {
			return types.ConnectionUpdatedMsg{Err: nil}
		}
		conn, err := c.config.UpdateConnection(id, name, host, port, password, dbNum, useCluster)
		if err != nil {
			slog.Error("Failed to update connection", "error", err)
		}
		return types.ConnectionUpdatedMsg{Connection: conn, Err: err}
	}
}

func (c *Commands) DeleteConnection(id int64) tea.Cmd {
	return func() tea.Msg {
		if c.config == nil {
			return types.ConnectionDeletedMsg{Err: nil}
		}
		err := c.config.DeleteConnection(id)
		return types.ConnectionDeletedMsg{ID: id, Err: err}
	}
}

func (c *Commands) Connect(host string, port int, password string, dbNum int, useCluster bool) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ConnectedMsg{Err: nil}
		}
		var err error
		if useCluster {
			err = c.redis.ConnectCluster([]string{fmt.Sprintf("%s:%d", host, port)}, password)
		} else {
			err = c.redis.Connect(host, port, password, dbNum)
		}
		if err != nil {
			slog.Error("Failed to connect", "error", err)
		}
		return types.ConnectedMsg{Err: err}
	}
}

func (c *Commands) AutoConnect(conn types.Connection) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ConnectedMsg{Err: nil}
		}
		var err error
		if conn.UseCluster {
			err = c.redis.ConnectCluster([]string{fmt.Sprintf("%s:%d", conn.Host, conn.Port)}, conn.Password)
		} else if conn.UseTLS {
			if conn.TLSConfig == nil {
				return types.ConnectedMsg{Err: fmt.Errorf("TLS requested but TLS configuration is missing")}
			}
			tlsCfg, tlsErr := conn.TLSConfig.BuildTLSConfig()
			if tlsErr != nil {
				slog.Error("Failed to build TLS config", "error", tlsErr)
				return types.ConnectedMsg{Err: tlsErr}
			}
			err = c.redis.ConnectWithTLS(conn.Host, conn.Port, conn.Password, conn.DB, tlsCfg)
		} else {
			err = c.redis.Connect(conn.Host, conn.Port, conn.Password, conn.DB)
		}
		if err != nil {
			slog.Error("Failed to connect", "error", err)
		}
		return types.ConnectedMsg{Err: err}
	}
}

func (c *Commands) Disconnect() tea.Cmd {
	return func() tea.Msg {
		if c.redis != nil {
			_ = c.redis.Disconnect()
		}
		return types.DisconnectedMsg{}
	}
}

// Key commands

func (c *Commands) LoadKeys(pattern string, cursor uint64, count int64) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.KeysLoadedMsg{Err: nil}
		}
		keys, nextCursor, err := c.redis.ScanKeys(pattern, cursor, count)
		totalKeys := c.redis.GetTotalKeys()
		return types.KeysLoadedMsg{Keys: keys, Cursor: nextCursor, TotalKeys: totalKeys, Err: err}
	}
}

func (c *Commands) LoadKeyValue(key string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.KeyValueLoadedMsg{Err: nil}
		}
		value, err := c.redis.GetValue(key)
		return types.KeyValueLoadedMsg{Key: key, Value: value, Err: err}
	}
}

func (c *Commands) LoadKeyPreview(key string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.KeyPreviewLoadedMsg{Err: nil}
		}
		value, err := c.redis.GetValue(key)
		return types.KeyPreviewLoadedMsg{Key: key, Value: value, Err: err}
	}
}

func (c *Commands) DeleteKey(key string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.KeyDeletedMsg{Key: key, Err: nil}
		}
		err := c.redis.DeleteKey(key)
		return types.KeyDeletedMsg{Key: key, Err: err}
	}
}

func (c *Commands) SetTTL(key string, ttl time.Duration) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.TTLSetMsg{Key: key, Err: nil}
		}
		err := c.redis.SetTTL(key, ttl)
		return types.TTLSetMsg{Key: key, TTL: ttl, Err: err}
	}
}

func (c *Commands) CreateKey(key string, keyType types.KeyType, value string, extra string, ttl time.Duration) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.KeySetMsg{Key: key, Err: nil}
		}

		// Delete existing key to prevent WRONGTYPE errors
		_ = c.redis.DeleteKey(key)

		var err error
		switch keyType {
		case types.KeyTypeString:
			err = c.redis.SetString(key, value, ttl)
		case types.KeyTypeList:
			err = c.redis.RPush(key, value)
		case types.KeyTypeSet:
			err = c.redis.SAdd(key, value)
		case types.KeyTypeZSet:
			score := 0.0
			if extra != "" {
				score, _ = strconv.ParseFloat(extra, 64)
			}
			err = c.redis.ZAdd(key, score, value)
		case types.KeyTypeHash:
			field := extra
			if field == "" {
				field = "field"
			}
			err = c.redis.HSet(key, field, value)
		case types.KeyTypeStream:
			field := extra
			if field == "" {
				field = "data"
			}
			fields := map[string]interface{}{field: value}
			_, err = c.redis.XAdd(key, fields)
		}
		return types.KeySetMsg{Key: key, Err: err}
	}
}

// Edit commands

func (c *Commands) EditStringValue(key, value string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ValueEditedMsg{Key: key, Err: nil}
		}
		err := c.redis.SetString(key, value, 0)
		return types.ValueEditedMsg{Key: key, Err: err}
	}
}

func (c *Commands) EditListElement(key string, index int64, value string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ValueEditedMsg{Key: key, Err: nil}
		}
		err := c.redis.LSet(key, index, value)
		return types.ValueEditedMsg{Key: key, Err: err}
	}
}

func (c *Commands) EditHashField(key, field, value string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ValueEditedMsg{Key: key, Err: nil}
		}
		err := c.redis.HSet(key, field, value)
		return types.ValueEditedMsg{Key: key, Err: err}
	}
}

// Collection commands

func (c *Commands) AddToList(key string, values ...string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ItemAddedToCollectionMsg{Key: key, Err: nil}
		}
		err := c.redis.RPush(key, values...)
		return types.ItemAddedToCollectionMsg{Key: key, Err: err}
	}
}

func (c *Commands) AddToSet(key string, members ...string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ItemAddedToCollectionMsg{Key: key, Err: nil}
		}
		err := c.redis.SAdd(key, members...)
		return types.ItemAddedToCollectionMsg{Key: key, Err: err}
	}
}

func (c *Commands) AddToZSet(key string, score float64, member string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ItemAddedToCollectionMsg{Key: key, Err: nil}
		}
		err := c.redis.ZAdd(key, score, member)
		return types.ItemAddedToCollectionMsg{Key: key, Err: err}
	}
}

func (c *Commands) AddToHash(key, field, value string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ItemAddedToCollectionMsg{Key: key, Err: nil}
		}
		err := c.redis.HSet(key, field, value)
		return types.ItemAddedToCollectionMsg{Key: key, Err: err}
	}
}

func (c *Commands) AddToStream(key string, fields map[string]interface{}) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ItemAddedToCollectionMsg{Key: key, Err: nil}
		}
		_, err := c.redis.XAdd(key, fields)
		return types.ItemAddedToCollectionMsg{Key: key, Err: err}
	}
}

func (c *Commands) RemoveFromList(key string, value string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ItemRemovedFromCollectionMsg{Key: key, Err: nil}
		}
		err := c.redis.LRem(key, 1, value)
		return types.ItemRemovedFromCollectionMsg{Key: key, Err: err}
	}
}

func (c *Commands) RemoveFromSet(key string, members ...string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ItemRemovedFromCollectionMsg{Key: key, Err: nil}
		}
		err := c.redis.SRem(key, members...)
		return types.ItemRemovedFromCollectionMsg{Key: key, Err: err}
	}
}

func (c *Commands) RemoveFromZSet(key string, members ...string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ItemRemovedFromCollectionMsg{Key: key, Err: nil}
		}
		err := c.redis.ZRem(key, members...)
		return types.ItemRemovedFromCollectionMsg{Key: key, Err: err}
	}
}

func (c *Commands) RemoveFromHash(key string, fields ...string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ItemRemovedFromCollectionMsg{Key: key, Err: nil}
		}
		err := c.redis.HDel(key, fields...)
		return types.ItemRemovedFromCollectionMsg{Key: key, Err: err}
	}
}

func (c *Commands) RemoveFromStream(key string, ids ...string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ItemRemovedFromCollectionMsg{Key: key, Err: nil}
		}
		err := c.redis.XDel(key, ids...)
		return types.ItemRemovedFromCollectionMsg{Key: key, Err: err}
	}
}

// Key operations

func (c *Commands) RenameKey(oldKey, newKey string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.KeyRenamedMsg{OldKey: oldKey, NewKey: newKey, Err: nil}
		}
		err := c.redis.Rename(oldKey, newKey)
		return types.KeyRenamedMsg{OldKey: oldKey, NewKey: newKey, Err: err}
	}
}

func (c *Commands) CopyKey(src, dst string, replace bool) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.KeyCopiedMsg{SourceKey: src, DestKey: dst, Err: nil}
		}
		err := c.redis.Copy(src, dst, replace)
		return types.KeyCopiedMsg{SourceKey: src, DestKey: dst, Err: err}
	}
}

func (c *Commands) SwitchDB(dbNum int) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.DBSwitchedMsg{DB: dbNum, Err: nil}
		}
		err := c.redis.SelectDB(dbNum)
		return types.DBSwitchedMsg{DB: dbNum, Err: err}
	}
}

// Server commands

func (c *Commands) LoadServerInfo() tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ServerInfoLoadedMsg{Err: nil}
		}
		info, err := c.redis.GetServerInfo()
		return types.ServerInfoLoadedMsg{Info: info, Err: err}
	}
}

func (c *Commands) FlushDB() tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.FlushDBMsg{Err: nil}
		}
		err := c.redis.FlushDB()
		return types.FlushDBMsg{Err: err}
	}
}

func (c *Commands) GetMemoryUsage(key string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.MemoryUsageMsg{Key: key, Err: nil}
		}
		bytes, err := c.redis.MemoryUsage(key)
		return types.MemoryUsageMsg{Key: key, Bytes: bytes, Err: err}
	}
}

func (c *Commands) GetSlowLog(count int64) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.SlowLogLoadedMsg{Err: nil}
		}
		entries, err := c.redis.SlowLogGet(count)
		return types.SlowLogLoadedMsg{Entries: entries, Err: err}
	}
}

func (c *Commands) GetClientList() tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ClientListLoadedMsg{Err: nil}
		}
		clients, err := c.redis.ClientList()
		return types.ClientListLoadedMsg{Clients: clients, Err: err}
	}
}

func (c *Commands) GetMemoryStats() tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.MemoryStatsLoadedMsg{Err: nil}
		}
		stats, err := c.redis.GetMemoryStats()
		return types.MemoryStatsLoadedMsg{Stats: stats, Err: err}
	}
}

func (c *Commands) GetClusterInfo() tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ClusterInfoLoadedMsg{Err: nil}
		}
		nodes, err := c.redis.ClusterNodes()
		info, _ := c.redis.ClusterInfo()
		return types.ClusterInfoLoadedMsg{Nodes: nodes, Info: info, Err: err}
	}
}

// Search commands

func (c *Commands) SearchByValue(pattern, valueSearch string, maxKeys int) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.KeysLoadedMsg{Err: nil}
		}
		keys, err := c.redis.SearchByValue(pattern, valueSearch, maxKeys)
		return types.KeysLoadedMsg{Keys: keys, Cursor: 0, Err: err}
	}
}

func (c *Commands) RegexSearch(pattern string, maxKeys int) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.RegexSearchResultMsg{Err: nil}
		}
		keys, err := c.redis.ScanKeysWithRegex(pattern, maxKeys)
		return types.RegexSearchResultMsg{Keys: keys, Err: err}
	}
}

func (c *Commands) FuzzySearch(term string, maxKeys int) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.FuzzySearchResultMsg{Err: nil}
		}
		keys, err := c.redis.FuzzySearchKeys(term, maxKeys)
		return types.FuzzySearchResultMsg{Keys: keys, Err: err}
	}
}

func (c *Commands) CompareKeys(key1, key2 string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.CompareKeysResultMsg{Err: nil}
		}
		val1, val2, err := c.redis.CompareKeys(key1, key2)
		return types.CompareKeysResultMsg{Key1Value: val1, Key2Value: val2, Err: err}
	}
}

// Bulk operations

func (c *Commands) BulkDelete(pattern string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.BulkDeleteMsg{Pattern: pattern, Err: nil}
		}
		deleted, err := c.redis.BulkDelete(pattern)
		return types.BulkDeleteMsg{Pattern: pattern, Deleted: deleted, Err: err}
	}
}

func (c *Commands) BatchSetTTL(pattern string, ttl time.Duration) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.BatchTTLSetMsg{Pattern: pattern, Err: nil}
		}
		count, err := c.redis.BatchSetTTL(pattern, ttl)
		return types.BatchTTLSetMsg{Pattern: pattern, Count: count, TTL: ttl, Err: err}
	}
}

// Script commands

func (c *Commands) EvalLuaScript(script string, keys []string, args ...interface{}) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.LuaScriptResultMsg{Err: nil}
		}
		result, err := c.redis.Eval(script, keys, args...)
		return types.LuaScriptResultMsg{Result: result, Err: err}
	}
}

// Pub/Sub commands

func (c *Commands) PublishMessage(channel, message string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.PublishResultMsg{Channel: channel, Err: nil}
		}
		receivers, err := c.redis.Publish(channel, message)
		return types.PublishResultMsg{Channel: channel, Receivers: receivers, Err: err}
	}
}

func (c *Commands) GetPubSubChannels(pattern string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ServerInfoLoadedMsg{Err: nil}
		}
		channels, _ := c.redis.PubSubChannels(pattern)
		info := types.ServerInfo{}
		for _, ch := range channels {
			if info.ClusterInfo != "" {
				info.ClusterInfo += ", "
			}
			info.ClusterInfo += ch
		}
		return types.ServerInfoLoadedMsg{Info: info, Err: nil}
	}
}

// Export/Import commands

func (c *Commands) ExportKeys(pattern, filename string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ExportCompleteMsg{Filename: filename, Err: nil}
		}
		data, err := c.redis.ExportKeys(pattern)
		if err != nil {
			return types.ExportCompleteMsg{Filename: filename, Err: err}
		}

		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return types.ExportCompleteMsg{Filename: filename, Err: err}
		}

		err = os.WriteFile(filename, jsonData, 0600)
		return types.ExportCompleteMsg{Filename: filename, KeyCount: len(data), Err: err}
	}
}

func (c *Commands) ImportKeys(filename string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ImportCompleteMsg{Filename: filename, Err: nil}
		}

		cleanPath := filepath.Clean(filename)
		jsonData, err := os.ReadFile(cleanPath) // #nosec G304 - user-provided import path is intentional
		if err != nil {
			return types.ImportCompleteMsg{Filename: filename, Err: err}
		}

		var data map[string]interface{}
		if err := json.Unmarshal(jsonData, &data); err != nil {
			return types.ImportCompleteMsg{Filename: filename, Err: err}
		}

		count, err := c.redis.ImportKeys(data)
		return types.ImportCompleteMsg{Filename: filename, KeyCount: count, Err: err}
	}
}

func (c *Commands) TestConnection(host string, port int, password string, db int) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ConnectionTestMsg{Success: false, Err: nil}
		}
		latency, err := c.redis.TestConnection(host, port, password, db)
		return types.ConnectionTestMsg{Success: err == nil, Latency: latency, Err: err}
	}
}

// Favorites commands

func (c *Commands) LoadFavorites(connID int64) tea.Cmd {
	return func() tea.Msg {
		if c.config == nil {
			return types.FavoritesLoadedMsg{Err: nil}
		}
		favorites := c.config.ListFavorites(connID)
		return types.FavoritesLoadedMsg{Favorites: favorites, Err: nil}
	}
}

func (c *Commands) AddFavorite(connID int64, key, label string) tea.Cmd {
	return func() tea.Msg {
		if c.config == nil {
			return types.FavoriteAddedMsg{Err: nil}
		}
		fav, err := c.config.AddFavorite(connID, key, label)
		return types.FavoriteAddedMsg{Favorite: fav, Err: err}
	}
}

func (c *Commands) RemoveFavorite(connID int64, key string) tea.Cmd {
	return func() tea.Msg {
		if c.config == nil {
			return types.FavoriteRemovedMsg{Err: nil}
		}
		err := c.config.RemoveFavorite(connID, key)
		return types.FavoriteRemovedMsg{Key: key, Err: err}
	}
}

// Recent keys commands

func (c *Commands) LoadRecentKeys(connID int64) tea.Cmd {
	return func() tea.Msg {
		if c.config == nil {
			return types.RecentKeysLoadedMsg{Err: nil}
		}
		keys := c.config.ListRecentKeys(connID)
		return types.RecentKeysLoadedMsg{Keys: keys, Err: nil}
	}
}

func (c *Commands) AddRecentKey(connID int64, key string, keyType types.KeyType) tea.Cmd {
	return func() tea.Msg {
		if c.config != nil {
			c.config.AddRecentKey(connID, key, keyType)
		}
		return nil
	}
}

// Templates commands

func (c *Commands) LoadTemplates() tea.Cmd {
	return func() tea.Msg {
		if c.config == nil {
			return types.TemplatesLoadedMsg{Err: nil}
		}
		templates := c.config.ListTemplates()
		return types.TemplatesLoadedMsg{Templates: templates, Err: nil}
	}
}

// Value history commands

func (c *Commands) LoadValueHistory(key string) tea.Cmd {
	return func() tea.Msg {
		if c.config == nil {
			return types.ValueHistoryMsg{Err: nil}
		}
		history := c.config.GetValueHistory(key)
		return types.ValueHistoryMsg{History: history, Err: nil}
	}
}

func (c *Commands) SaveValueHistory(key string, value types.RedisValue, action string) tea.Cmd {
	return func() tea.Msg {
		if c.config != nil {
			c.config.AddValueHistory(key, value, action)
		}
		return nil
	}
}

// Keyspace events commands

func (c *Commands) SubscribeKeyspace(pattern string, sendFunc func(tea.Msg)) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.KeyspaceSubscribedMsg{Err: nil}
		}
		err := c.redis.SubscribeKeyspace(pattern, func(event types.KeyspaceEvent) {
			if sendFunc != nil {
				sendFunc(types.KeyspaceEventMsg{Event: event})
			}
		})
		return types.KeyspaceSubscribedMsg{Pattern: pattern, Err: err}
	}
}

func (c *Commands) UnsubscribeKeyspace() tea.Cmd {
	return func() tea.Msg {
		if c.redis != nil {
			_ = c.redis.UnsubscribeKeyspace()
		}
		return nil
	}
}

// Tree view commands

func (c *Commands) LoadKeyPrefixes(separator string, maxDepth int) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.TreeNodeExpandedMsg{Err: nil}
		}
		prefixes, err := c.redis.GetKeyPrefixes(separator, maxDepth)
		return types.TreeNodeExpandedMsg{Children: prefixes, Err: err}
	}
}

// Utility commands (no dependencies)

func (c *Commands) WatchKeyTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return types.WatchTickMsg{}
	})
}

func (c *Commands) CopyToClipboard(content string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(content)
		err := cmd.Run()
		return types.ClipboardCopiedMsg{Content: content, Err: err}
	}
}
