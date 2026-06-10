package cmd

import (
	"github.com/davidbudnick/redis-tui/internal/types"

	tea "charm.land/bubbletea/v2"
)

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

func (c *Commands) LoadTemplates() tea.Cmd {
	return func() tea.Msg {
		if c.config == nil {
			return types.TemplatesLoadedMsg{Err: nil}
		}
		templates := c.config.ListTemplates()
		return types.TemplatesLoadedMsg{Templates: templates, Err: nil}
	}
}

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

func (c *Commands) LoadRedisConfig(pattern string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ConfigLoadedMsg{Err: nil}
		}
		params, err := c.redis.ConfigGet(pattern)
		return types.ConfigLoadedMsg{Params: params, Err: err}
	}
}

func (c *Commands) SetRedisConfig(param, value string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ConfigSetMsg{Param: param, Value: value, Err: nil}
		}
		err := c.redis.ConfigSet(param, value)
		return types.ConfigSetMsg{Param: param, Value: value, Err: err}
	}
}
