package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"

	tea "charm.land/bubbletea/v2"
)

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

		var data map[string]any
		if err := json.Unmarshal(jsonData, &data); err != nil {
			return types.ImportCompleteMsg{Filename: filename, Err: err}
		}

		count, err := c.redis.ImportKeys(data)
		return types.ImportCompleteMsg{Filename: filename, KeyCount: count, Err: err}
	}
}

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

func (c *Commands) EvalLuaScript(script string, keys []string, args ...any) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.LuaScriptResultMsg{Err: nil}
		}
		result, err := c.redis.Eval(script, keys, args...)
		return types.LuaScriptResultMsg{Result: result, Err: err}
	}
}

func (c *Commands) JSONPathQuery(key, path string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.JSONPathResultMsg{Err: nil}
		}
		result, err := c.redis.JSONGetPath(key, path)
		return types.JSONPathResultMsg{Result: result, Err: err}
	}
}
