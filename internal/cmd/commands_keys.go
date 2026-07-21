package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/davidbudnick/redis-tui/internal/types"

	tea "charm.land/bubbletea/v2"
)

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
		value, err := c.redis.GetValuePreview(key)
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
				var parseErr error
				score, parseErr = strconv.ParseFloat(extra, 64)
				if parseErr != nil {
					return types.KeySetMsg{Key: key, Err: fmt.Errorf("invalid score %q: %w", extra, parseErr)}
				}
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
			fields := map[string]any{field: value}
			_, err = c.redis.XAdd(key, fields)
		case types.KeyTypeJSON:
			err = c.redis.JSONSet(key, value)
		}
		return types.KeySetMsg{Key: key, Err: err}
	}
}

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

func (c *Commands) EditJSONValue(key, value string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ValueEditedMsg{Key: key, Err: nil}
		}
		err := c.redis.JSONSet(key, value)
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
