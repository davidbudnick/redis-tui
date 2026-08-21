package ui

import (
	"github.com/davidbudnick/redis-tui/internal/types"

	tea "charm.land/bubbletea/v2"
)

func (m Model) handleConnectionsScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.SelectedConnIdx > 0 {
			m.SelectedConnIdx--
			m.ConnectionError = "" // Clear error on navigation
		}
	case "down", "j":
		if m.SelectedConnIdx < len(m.Connections)-1 {
			m.SelectedConnIdx++
			m.ConnectionError = "" // Clear error on navigation
		}
	case "enter":
		if len(m.Connections) > 0 && m.SelectedConnIdx < len(m.Connections) {
			conn := m.Connections[m.SelectedConnIdx]
			m.CurrentConn = &conn
			m.Loading = true
			m.StatusMsg = "Connecting..."
			m.ConnectionError = "" // Clear any previous connection error
			return m, m.Cmds.Connect(conn)
		}
	case "a", "n":
		m.Screen = types.ScreenAddConnection
		m.resetConnInputs()
	case "e":
		if len(m.Connections) > 0 && m.SelectedConnIdx < len(m.Connections) {
			conn := m.Connections[m.SelectedConnIdx]
			m.EditingConnection = &conn
			m.populateConnInputs(conn)
			m.Screen = types.ScreenEditConnection
		}
	case "d", "delete", "backspace":
		if len(m.Connections) > 0 && m.SelectedConnIdx < len(m.Connections) {
			m.ConfirmType = "connection"
			m.ConfirmData = m.Connections[m.SelectedConnIdx]
			m.Screen = types.ScreenConfirmDelete
		}
	case "r":
		m.Loading = true
		return m, m.Cmds.LoadConnections()
	}
	return m, nil
}

func (m Model) handleAddConnectionScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	fieldCount := m.connFieldCount()

	switch msg.String() {
	case "tab", "down":
		m.blurConnField()
		m.ConnFocusIdx = (m.ConnFocusIdx + 1) % fieldCount
		m.focusConnField()
	case "shift+tab", "up":
		m.blurConnField()
		m.ConnFocusIdx--
		if m.ConnFocusIdx < 0 {
			m.ConnFocusIdx = fieldCount - 1
		}
		m.focusConnField()
	case "space":
		if m.ConnFocusIdx == 8 {
			m.ConnClusterMode = !m.ConnClusterMode
			return m, nil
		}
		return m.updateConnInputs(msg)
	case "enter":
		if m.ConnFocusIdx == 8 {
			m.ConnClusterMode = !m.ConnClusterMode
			return m, nil
		}
		if m.ConnInputs[0].Value() != "" && m.ConnInputs[1].Value() != "" {
			m.Loading = true
			conn := m.convertCurrentInputsToConnection(m.ConnInputs, "add")
			return m, m.Cmds.AddConnection(
				conn,
			)
		}
	case "ctrl+t":
		m.Loading = true
		m.Screen = types.ScreenTestConnection
		conn := m.convertCurrentInputsToConnection(m.ConnInputs, "test")
		return m, m.Cmds.TestConnection(
			conn,
		)
	case "esc":
		m.Screen = types.ScreenConnections
		m.resetConnInputs()
	default:
		return m.updateConnInputs(msg)
	}
	return m, nil
}

// connInputIndex maps a ConnFocusIdx to the actual ConnInputs array index.
// Indices 0-7 map directly to ConnInputs[0-7], index 8 is the cluster toggle,
// and index 9 maps to ConnInputs[8] (Database).
func connInputIndex(focusIdx int) int {
	if focusIdx <= 7 {
		return focusIdx
	}
	if focusIdx == 9 {
		return 8 // Database input
	}
	return -1 // cluster toggle, no text input
}

func (m *Model) blurConnField() {
	idx := connInputIndex(m.ConnFocusIdx)
	if idx >= 0 && idx < len(m.ConnInputs) {
		m.ConnInputs[idx].Blur()
	}
}

func (m *Model) focusConnField() {
	idx := connInputIndex(m.ConnFocusIdx)
	if idx >= 0 && idx < len(m.ConnInputs) {
		m.ConnInputs[idx].Focus()
	}
}

func (m Model) updateConnInputs(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Only update the focused text input, not the cluster toggle
	idx := connInputIndex(m.ConnFocusIdx)
	if idx >= 0 && idx < len(m.ConnInputs) {
		var inputCmd tea.Cmd
		m.ConnInputs[idx], inputCmd = m.ConnInputs[idx].Update(msg)
		return m, inputCmd
	}
	return m, nil
}

func (m Model) handleEditConnectionScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	fieldCount := m.connFieldCount()

	switch msg.String() {
	case "tab", "down":
		m.blurConnField()
		m.ConnFocusIdx = (m.ConnFocusIdx + 1) % fieldCount
		m.focusConnField()
	case "shift+tab", "up":
		m.blurConnField()
		m.ConnFocusIdx--
		if m.ConnFocusIdx < 0 {
			m.ConnFocusIdx = fieldCount - 1
		}
		m.focusConnField()
	case "space":
		if m.ConnFocusIdx == 8 {
			m.ConnClusterMode = !m.ConnClusterMode
			return m, nil
		}
		return m.updateConnInputs(msg)
	case "enter":
		if m.ConnFocusIdx == 8 {
			m.ConnClusterMode = !m.ConnClusterMode
			return m, nil
		}
		if m.EditingConnection != nil && m.ConnInputs[0].Value() != "" && m.ConnInputs[1].Value() != "" {
			m.Loading = true
			conn := m.convertCurrentInputsToConnection(m.ConnInputs, "edit")
			return m, m.Cmds.UpdateConnection(
				conn,
			)
		}
	case "esc":
		m.Screen = types.ScreenConnections
		m.EditingConnection = nil
		m.resetConnInputs()
	default:
		return m.updateConnInputs(msg)
	}
	return m, nil
}

func (m Model) handleTestConnectionScreen(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.Screen = types.ScreenAddConnection
	}
	return m, nil
}
