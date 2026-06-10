package cmd

import (
	"github.com/davidbudnick/redis-tui/internal/types"

	tea "charm.land/bubbletea/v2"
)

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

func (c *Commands) LoadKeyPrefixes(separator string, maxDepth int) tea.Cmd {
	return func() tea.Msg {
		if c.redis == nil {
			return types.TreeNodeExpandedMsg{Err: nil}
		}
		prefixes, err := c.redis.GetKeyPrefixes(separator, maxDepth)
		return types.TreeNodeExpandedMsg{Children: prefixes, Err: err}
	}
}
