package ui

import (
	"github.com/davidbudnick/redis-tui/internal/types"

	tea "charm.land/bubbletea/v2"
)

func (m Model) handleFavoritesScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.SelectedFavIdx > 0 {
			m.SelectedFavIdx--
		}
	case "down", "j":
		if m.SelectedFavIdx < len(m.Favorites)-1 {
			m.SelectedFavIdx++
		}
	case "enter":
		if len(m.Favorites) > 0 && m.SelectedFavIdx < len(m.Favorites) {
			key := m.Favorites[m.SelectedFavIdx].Key
			for i, k := range m.Keys {
				if k.Key == key {
					m.SelectedKeyIdx = i
					m.CurrentKey = &m.Keys[i]
					m.Screen = types.ScreenKeyDetail
					return m, m.Cmds.LoadKeyValue(key)
				}
			}
		}
	case "d":
		if len(m.Favorites) > 0 && m.SelectedFavIdx < len(m.Favorites) {
			return m, m.Cmds.RemoveFavorite(m.Favorites[m.SelectedFavIdx].ConnectionID, m.Favorites[m.SelectedFavIdx].Key)
		}
	case "esc":
		m.Screen = types.ScreenKeys
	}
	return m, nil
}

func (m Model) handleRecentKeysScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.SelectedRecentIdx > 0 {
			m.SelectedRecentIdx--
		}
	case "down", "j":
		if m.SelectedRecentIdx < len(m.RecentKeys)-1 {
			m.SelectedRecentIdx++
		}
	case "enter":
		if len(m.RecentKeys) > 0 && m.SelectedRecentIdx < len(m.RecentKeys) {
			key := m.RecentKeys[m.SelectedRecentIdx].Key
			for i, k := range m.Keys {
				if k.Key == key {
					m.SelectedKeyIdx = i
					m.CurrentKey = &m.Keys[i]
					m.Screen = types.ScreenKeyDetail
					return m, m.Cmds.LoadKeyValue(key)
				}
			}
		}
	case "esc":
		m.Screen = types.ScreenKeys
	}
	return m, nil
}

func (m Model) handleTreeViewScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.SelectedTreeIdx > 0 {
			m.SelectedTreeIdx--
		}
	case "down", "j":
		if m.SelectedTreeIdx < len(m.TreeNodes)-1 {
			m.SelectedTreeIdx++
		}
	case "enter", "space":
		if len(m.TreeNodes) > 0 && m.SelectedTreeIdx < len(m.TreeNodes) {
			node := m.TreeNodes[m.SelectedTreeIdx]
			if !node.IsKey {
				m.TreeExpanded[node.FullPath] = !m.TreeExpanded[node.FullPath]
			} else {
				// Navigate to key
				for i, k := range m.Keys {
					if k.Key == node.FullPath {
						m.SelectedKeyIdx = i
						m.CurrentKey = &m.Keys[i]
						m.Screen = types.ScreenKeyDetail
						return m, m.Cmds.LoadKeyValue(node.FullPath)
					}
				}
			}
		}
	case "esc":
		m.Screen = types.ScreenKeys
	}
	return m, nil
}

func (m Model) handleTemplatesScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.SelectedTemplateIdx > 0 {
			m.SelectedTemplateIdx--
		}
	case "down", "j":
		if m.SelectedTemplateIdx < len(m.Templates)-1 {
			m.SelectedTemplateIdx++
		}
	case "enter":
		// Use template to create a key
		if len(m.Templates) > 0 && m.SelectedTemplateIdx < len(m.Templates) {
			template := m.Templates[m.SelectedTemplateIdx]
			m.AddKeyInputs[0].SetValue(template.KeyPattern)
			m.AddKeyInputs[1].SetValue(template.DefaultValue)
			m.AddKeyType = template.Type
			m.Screen = types.ScreenAddKey
		}
	case "esc":
		m.Screen = types.ScreenKeys
	}
	return m, nil
}

func (m Model) handleValueHistoryScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.SelectedHistoryIdx > 0 {
			m.SelectedHistoryIdx--
		}
	case "down", "j":
		if m.SelectedHistoryIdx < len(m.ValueHistory)-1 {
			m.SelectedHistoryIdx++
		}
	case "enter":
		// Restore this value
		if m.CurrentKey != nil && len(m.ValueHistory) > 0 && m.SelectedHistoryIdx < len(m.ValueHistory) {
			entry := m.ValueHistory[m.SelectedHistoryIdx]
			m.Loading = true
			return m, m.Cmds.EditStringValue(m.CurrentKey.Key, entry.Value.StringValue)
		}
	case "esc":
		m.Screen = types.ScreenKeyDetail
	}
	return m, nil
}

func (m Model) handleKeyspaceEventsScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "c":
		m.KeyspaceEvents = nil
	case "esc":
		m.Screen = types.ScreenKeys
	}
	return m, nil
}

func (m Model) handleWatchKeyScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.WatchActive = false
		m.Screen = types.ScreenKeyDetail
	}
	return m, nil
}

func (m Model) handleConnectionGroupsScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.SelectedGroupIdx > 0 {
			m.SelectedGroupIdx--
		}
	case "down", "j":
		if m.SelectedGroupIdx < len(m.ConnectionGroups)-1 {
			m.SelectedGroupIdx++
		}
	case "esc":
		m.Screen = types.ScreenConnections
	}
	return m, nil
}

func (m Model) handleExpiringKeysScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.SelectedKeyIdx > 0 {
			m.SelectedKeyIdx--
		}
	case "down", "j":
		if len(m.ExpiringKeys) > 0 && m.SelectedKeyIdx < len(m.ExpiringKeys)-1 {
			m.SelectedKeyIdx++
		}
	case "enter":
		if len(m.ExpiringKeys) > 0 && m.SelectedKeyIdx < len(m.ExpiringKeys) {
			key := m.ExpiringKeys[m.SelectedKeyIdx]
			m.CurrentKey = &key
			m.Screen = types.ScreenKeyDetail
			return m, m.Cmds.LoadKeyValue(key.Key)
		}
	case "esc":
		m.Screen = types.ScreenKeys
	}
	return m, nil
}
