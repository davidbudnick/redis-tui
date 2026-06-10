package ui

import (
	"fmt"
	"strconv"

	"github.com/davidbudnick/redis-tui/internal/types"

	tea "charm.land/bubbletea/v2"
)

// Import/Export handlers

func (m Model) handleExportCompleteMsg(msg types.ExportCompleteMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.StatusMsg = "Export failed: " + msg.Err.Error()
	} else {
		m.StatusMsg = "Exported " + strconv.Itoa(msg.KeyCount) + " keys to " + msg.Filename
		m.Screen = types.ScreenKeys
	}
	return m, nil
}

func (m Model) handleImportCompleteMsg(msg types.ImportCompleteMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.StatusMsg = "Import failed: " + msg.Err.Error()
		return m, nil
	}
	m.StatusMsg = "Imported " + strconv.Itoa(msg.KeyCount) + " keys from " + msg.Filename
	m.Screen = types.ScreenKeys
	m.KeyCursor = 0
	return m, m.Cmds.LoadKeys(m.KeyPattern, 0, m.ScanSize)
}

// Feature message handlers

func (m Model) handleBulkDeleteMsg(msg types.BulkDeleteMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.StatusMsg = "Bulk delete error: " + msg.Err.Error()
		return m, nil
	}
	m.StatusMsg = "Deleted " + strconv.Itoa(msg.Deleted) + " keys"
	m.Screen = types.ScreenKeys
	m.KeyCursor = 0
	return m, m.Cmds.LoadKeys(m.KeyPattern, 0, m.ScanSize)
}

func (m Model) handleFavoritesLoadedMsg(msg types.FavoritesLoadedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err == nil {
		m.Favorites = msg.Favorites
	}
	return m, nil
}

func (m Model) handleFavoriteAddedMsg(msg types.FavoriteAddedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err == nil {
		m.StatusMsg = "Added to favorites"
		for i := range m.Keys {
			if m.Keys[i].Key == msg.Favorite.Key {
				m.Keys[i].IsFavorite = true
				break
			}
		}
		if m.CurrentKey != nil && m.CurrentKey.Key == msg.Favorite.Key {
			m.CurrentKey.IsFavorite = true
		}
	}
	return m, nil
}

func (m Model) handleFavoriteRemovedMsg(msg types.FavoriteRemovedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err == nil {
		m.StatusMsg = "Removed from favorites"
		for i := range m.Keys {
			if m.Keys[i].Key == msg.Key {
				m.Keys[i].IsFavorite = false
				break
			}
		}
		if m.CurrentKey != nil && m.CurrentKey.Key == msg.Key {
			m.CurrentKey.IsFavorite = false
		}
	}
	return m, nil
}

func (m Model) handleRecentKeysLoadedMsg(msg types.RecentKeysLoadedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err == nil {
		m.RecentKeys = msg.Keys
	}
	return m, nil
}

func (m Model) handleTemplatesLoadedMsg(msg types.TemplatesLoadedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err == nil {
		m.Templates = msg.Templates
		m.Screen = types.ScreenTemplates
	}
	return m, nil
}

func (m Model) handleValueHistoryMsg(msg types.ValueHistoryMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err == nil {
		m.ValueHistory = msg.History
		m.Screen = types.ScreenValueHistory
	}
	return m, nil
}

// Search message handlers

func (m Model) handleRegexSearchResultMsg(msg types.RegexSearchResultMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.StatusMsg = "Regex search error: " + msg.Err.Error()
	} else {
		m.Keys = msg.Keys
		m.Screen = types.ScreenKeys
		m.StatusMsg = "Found " + strconv.Itoa(len(msg.Keys)) + " keys"
	}
	return m, nil
}

func (m Model) handleFuzzySearchResultMsg(msg types.FuzzySearchResultMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.StatusMsg = "Fuzzy search error: " + msg.Err.Error()
	} else {
		m.Keys = msg.Keys
		m.Screen = types.ScreenKeys
		m.StatusMsg = "Found " + strconv.Itoa(len(msg.Keys)) + " keys"
	}
	return m, nil
}

func (m Model) handleCompareKeysResultMsg(msg types.CompareKeysResultMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.StatusMsg = "Compare error: " + msg.Err.Error()
	} else {
		equal := msg.Key1Value.StringValue == msg.Key2Value.StringValue
		m.CompareResult = &types.KeyComparison{
			Equal:       equal,
			Differences: []string{msg.Diff},
		}
	}
	return m, nil
}

func (m Model) handleJSONPathResultMsg(msg types.JSONPathResultMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.JSONPathResult = "Error: " + msg.Err.Error()
	} else {
		if s, ok := msg.Result.(string); ok {
			m.JSONPathResult = s
		} else {
			m.JSONPathResult = fmt.Sprintf("%v", msg.Result)
		}
	}
	return m, nil
}

func (m Model) handleClipboardCopiedMsg(msg types.ClipboardCopiedMsg) (tea.Model, tea.Cmd) {
	m.Loading = false
	if msg.Err != nil {
		m.StatusMsg = "Copy failed: " + msg.Err.Error()
	} else {
		m.StatusMsg = "Copied to clipboard"
	}
	return m, nil
}
