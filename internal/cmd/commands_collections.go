package cmd

import (
	"github.com/davidbudnick/redis-tui/internal/types"
	"github.com/redis/go-redis/v9"

	tea "charm.land/bubbletea/v2"
)

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

func (c *Commands) AddToStream(key string, fields map[string]any) tea.Cmd {
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

func (c *Commands) AddToHLL(key string, elements ...string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ItemAddedToCollectionMsg{Key: key, Err: nil}
		}
		err := c.redis.PFAdd(key, elements...)
		return types.ItemAddedToCollectionMsg{Key: key, Err: err}
	}
}

func (c *Commands) AddToGeo(key string, lon, lat float64, member string) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ItemAddedToCollectionMsg{Key: key, Err: nil}
		}
		err := c.redis.GeoAdd(key, &redis.GeoLocation{Name: member, Longitude: lon, Latitude: lat})
		return types.ItemAddedToCollectionMsg{Key: key, Err: err}
	}
}

func (c *Commands) SetBit(key string, offset int64, value int) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.ItemAddedToCollectionMsg{Key: key, Err: nil}
		}
		err := c.redis.SetBit(key, offset, value)
		return types.ItemAddedToCollectionMsg{Key: key, Err: err}
	}
}
