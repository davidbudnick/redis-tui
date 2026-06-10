package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/davidbudnick/redis-tui/internal/types"
)

func TestHandleFavoritesScreen(t *testing.T) {
	favs := []types.Favorite{{Key: "a", ConnectionID: 1}, {Key: "b", ConnectionID: 1}}

	t.Run("up", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Favorites = favs
		m.SelectedFavIdx = 1
		_, _ = m.handleFavoritesScreen(keyMsg('k'))
	})
	t.Run("down", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Favorites = favs
		_, _ = m.handleFavoritesScreen(keyMsg('j'))
	})
	t.Run("enter opens matching key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Favorites = favs
		m.Keys = []types.RedisKey{{Key: "a"}}
		_, cmd := m.handleFavoritesScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter no matching key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Favorites = favs
		m.Keys = []types.RedisKey{{Key: "other"}}
		_, cmd := m.handleFavoritesScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("enter empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleFavoritesScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
	})
	t.Run("d removes", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Favorites = favs
		_, cmd := m.handleFavoritesScreen(keyMsg('d'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("d empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleFavoritesScreen(keyMsg('d'))
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleFavoritesScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
	})
}

func TestHandleRecentKeysScreen(t *testing.T) {
	recents := []types.RecentKey{{Key: "a"}, {Key: "b"}}

	t.Run("up", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.RecentKeys = recents
		m.SelectedRecentIdx = 1
		_, _ = m.handleRecentKeysScreen(keyMsg('k'))
	})
	t.Run("down", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.RecentKeys = recents
		_, _ = m.handleRecentKeysScreen(keyMsg('j'))
	})
	t.Run("enter matching", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.RecentKeys = recents
		m.Keys = []types.RedisKey{{Key: "a"}}
		_, cmd := m.handleRecentKeysScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter not matching", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.RecentKeys = recents
		_, _ = m.handleRecentKeysScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
	})
	t.Run("enter empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleRecentKeysScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleRecentKeysScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
	})
}

func TestHandleTreeViewScreen(t *testing.T) {
	nodes := []types.TreeNode{
		{FullPath: "user:", IsKey: false},
		{FullPath: "user:1", IsKey: true},
	}

	t.Run("up", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.TreeNodes = nodes
		m.SelectedTreeIdx = 1
		_, _ = m.handleTreeViewScreen(keyMsg('k'))
	})
	t.Run("down", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.TreeNodes = nodes
		_, _ = m.handleTreeViewScreen(keyMsg('j'))
	})
	t.Run("enter expands folder", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.TreeNodes = nodes
		result, _ := m.handleTreeViewScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if !result.(Model).TreeExpanded["user:"] {
			t.Error("expected expanded")
		}
	})
	t.Run("space also expands", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.TreeNodes = nodes
		_, _ = m.handleTreeViewScreen(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	})
	t.Run("enter key navigates", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.TreeNodes = nodes
		m.SelectedTreeIdx = 1
		m.Keys = []types.RedisKey{{Key: "user:1"}}
		_, cmd := m.handleTreeViewScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter key no match", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.TreeNodes = nodes
		m.SelectedTreeIdx = 1
		_, _ = m.handleTreeViewScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
	})
	t.Run("enter empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleTreeViewScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleTreeViewScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
	})
}

func TestHandleTemplatesScreen(t *testing.T) {
	tmpls := []types.KeyTemplate{
		{Name: "t1", KeyPattern: "user:{id}", Type: types.KeyTypeString, DefaultValue: "val"},
	}

	t.Run("up", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Templates = tmpls
		m.SelectedTemplateIdx = 0
		_, _ = m.handleTemplatesScreen(keyMsg('k'))
	})
	t.Run("down", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Templates = append(tmpls, types.KeyTemplate{Name: "t2"})
		_, _ = m.handleTemplatesScreen(keyMsg('j'))
	})
	t.Run("enter uses template", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Templates = tmpls
		result, _ := m.handleTemplatesScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if result.(Model).Screen != types.ScreenAddKey {
			t.Errorf("expected ScreenAddKey, got %v", result.(Model).Screen)
		}
	})
	t.Run("enter empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleTemplatesScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleTemplatesScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
	})
}

func TestHandleValueHistoryScreen(t *testing.T) {
	hist := []types.ValueHistoryEntry{{Key: "a", Value: types.RedisValue{StringValue: "old"}}}

	t.Run("up", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ValueHistory = hist
		m.SelectedHistoryIdx = 0
		_, _ = m.handleValueHistoryScreen(keyMsg('k'))
	})
	t.Run("down", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ValueHistory = append(hist, types.ValueHistoryEntry{Key: "b"})
		_, _ = m.handleValueHistoryScreen(keyMsg('j'))
	})
	t.Run("enter restores", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentKey = &types.RedisKey{Key: "a"}
		m.ValueHistory = hist
		_, cmd := m.handleValueHistoryScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter no current", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ValueHistory = hist
		_, cmd := m.handleValueHistoryScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleValueHistoryScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
	})
}

func TestHandleKeyspaceEventsScreen(t *testing.T) {
	t.Run("c clears", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.KeyspaceEvents = []types.KeyspaceEvent{{Key: "a"}}
		result, _ := m.handleKeyspaceEventsScreen(keyMsg('c'))
		if len(result.(Model).KeyspaceEvents) != 0 {
			t.Error("expected cleared")
		}
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleKeyspaceEventsScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
	})
}

func TestHandleWatchKeyScreen(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.WatchActive = true
	result, _ := m.handleWatchKeyScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
	if result.(Model).WatchActive {
		t.Error("expected watch off")
	}
}

func TestHandleConnectionGroupsScreen(t *testing.T) {
	groups := []types.ConnectionGroup{{Name: "a"}, {Name: "b"}}

	t.Run("up", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ConnectionGroups = groups
		m.SelectedGroupIdx = 1
		_, _ = m.handleConnectionGroupsScreen(keyMsg('k'))
	})
	t.Run("down", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ConnectionGroups = groups
		_, _ = m.handleConnectionGroupsScreen(keyMsg('j'))
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleConnectionGroupsScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
	})
}

func TestHandleExpiringKeysScreen(t *testing.T) {
	keys := []types.RedisKey{{Key: "a"}, {Key: "b"}}

	t.Run("up", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ExpiringKeys = keys
		m.SelectedKeyIdx = 1
		_, _ = m.handleExpiringKeysScreen(keyMsg('k'))
	})
	t.Run("down", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ExpiringKeys = keys
		_, _ = m.handleExpiringKeysScreen(keyMsg('j'))
	})
	t.Run("enter opens", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ExpiringKeys = keys
		_, cmd := m.handleExpiringKeysScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("enter empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleExpiringKeysScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
	})
	t.Run("esc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleExpiringKeysScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
	})
}
