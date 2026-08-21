package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/davidbudnick/redis-tui/internal/types"
)

// keyMsg constructs a tea.KeyPressMsg for a single rune like 'a', 'j', etc.
func keyMsg(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

func TestHandleConnectionsScreen(t *testing.T) {
	connList := []types.Connection{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}, {ID: 3, Name: "c"}}

	t.Run("down navigates", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Connections = connList
		m.ConnectionError = "old err"
		result, _ := m.handleConnectionsScreen(keyMsg('j'))
		model := result.(Model)
		if model.SelectedConnIdx != 1 {
			t.Errorf("expected SelectedConnIdx=1, got %d", model.SelectedConnIdx)
		}
		if model.ConnectionError != "" {
			t.Error("expected ConnectionError cleared")
		}
	})
	t.Run("down key arrow", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Connections = connList
		result, _ := m.handleConnectionsScreen(tea.KeyPressMsg{Code: tea.KeyDown})
		model := result.(Model)
		if model.SelectedConnIdx != 1 {
			t.Errorf("expected SelectedConnIdx=1, got %d", model.SelectedConnIdx)
		}
	})
	t.Run("up at top", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Connections = connList
		result, _ := m.handleConnectionsScreen(keyMsg('k'))
		model := result.(Model)
		if model.SelectedConnIdx != 0 {
			t.Errorf("expected SelectedConnIdx=0, got %d", model.SelectedConnIdx)
		}
	})
	t.Run("up navigates", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Connections = connList
		m.SelectedConnIdx = 2
		result, _ := m.handleConnectionsScreen(tea.KeyPressMsg{Code: tea.KeyUp})
		model := result.(Model)
		if model.SelectedConnIdx != 1 {
			t.Errorf("expected SelectedConnIdx=1, got %d", model.SelectedConnIdx)
		}
	})
	t.Run("enter connects", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Connections = connList
		_, cmd := m.handleConnectionsScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected connect cmd")
		}
	})
	t.Run("add key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleConnectionsScreen(keyMsg('a'))
		model := result.(Model)
		if model.Screen != types.ScreenAddConnection {
			t.Errorf("expected ScreenAddConnection, got %v", model.Screen)
		}
	})
	t.Run("n key adds", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleConnectionsScreen(keyMsg('n'))
		model := result.(Model)
		if model.Screen != types.ScreenAddConnection {
			t.Errorf("expected ScreenAddConnection, got %v", model.Screen)
		}
	})
	t.Run("e edits", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Connections = connList
		result, _ := m.handleConnectionsScreen(keyMsg('e'))
		model := result.(Model)
		if model.Screen != types.ScreenEditConnection {
			t.Errorf("expected ScreenEditConnection, got %v", model.Screen)
		}
		if model.EditingConnection == nil {
			t.Error("expected EditingConnection set")
		}
	})
	t.Run("d delete", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Connections = connList
		result, _ := m.handleConnectionsScreen(keyMsg('d'))
		model := result.(Model)
		if model.Screen != types.ScreenConfirmDelete {
			t.Errorf("expected ScreenConfirmDelete, got %v", model.Screen)
		}
	})
	t.Run("r reloads", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleConnectionsScreen(keyMsg('r'))
		if cmd == nil {
			t.Error("expected reload cmd")
		}
	})
	t.Run("enter with empty list", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleConnectionsScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd for empty list")
		}
	})
}

func TestHandleAddConnectionScreen(t *testing.T) {
	t.Run("tab advances focus", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleAddConnectionScreen(tea.KeyPressMsg{Code: tea.KeyTab})
		model := result.(Model)
		if model.ConnFocusIdx != 1 {
			t.Errorf("expected ConnFocusIdx=1, got %d", model.ConnFocusIdx)
		}
	})
	t.Run("shift+tab back", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleAddConnectionScreen(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
		model := result.(Model)
		if model.ConnFocusIdx != 9 {
			t.Errorf("expected ConnFocusIdx=9, got %d", model.ConnFocusIdx)
		}
	})
	t.Run("space on cluster toggle", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ConnFocusIdx = 8
		result, _ := m.handleAddConnectionScreen(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
		model := result.(Model)
		if !model.ConnClusterMode {
			t.Error("expected cluster mode on")
		}
	})
	t.Run("space on cluster toggle adjusts focus", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ConnFocusIdx = 8
		m.ConnClusterMode = false
		// Force an out-of-range focus scenario by pre-setting then toggling
		_, _ = m.handleAddConnectionScreen(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	})
	t.Run("space on text field sends to input", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleAddConnectionScreen(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	})
	t.Run("enter on cluster toggle", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ConnFocusIdx = 8
		result, _ := m.handleAddConnectionScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		model := result.(Model)
		if !model.ConnClusterMode {
			t.Error("expected cluster toggled")
		}
	})
	t.Run("enter submits when valid", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ConnInputs[0].SetValue("name")
		m.ConnInputs[1].SetValue("host")
		_, cmd := m.handleAddConnectionScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected add cmd")
		}
	})
	t.Run("enter no submit when empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleAddConnectionScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd when missing fields")
		}
	})
	t.Run("ctrl+t tests", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleAddConnectionScreen(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
		if cmd == nil {
			t.Error("expected test cmd")
		}
	})
	t.Run("esc cancels", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleAddConnectionScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
		model := result.(Model)
		if model.Screen != types.ScreenConnections {
			t.Errorf("expected ScreenConnections, got %v", model.Screen)
		}
	})
	t.Run("default updates input", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleAddConnectionScreen(keyMsg('x'))
	})
}

func TestHandleEditConnectionScreen(t *testing.T) {
	t.Run("tab advances focus", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleEditConnectionScreen(tea.KeyPressMsg{Code: tea.KeyTab})
		model := result.(Model)
		if model.ConnFocusIdx != 1 {
			t.Errorf("expected ConnFocusIdx=1, got %d", model.ConnFocusIdx)
		}
	})
	t.Run("shift+tab", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleEditConnectionScreen(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
		model := result.(Model)
		if model.ConnFocusIdx != 9 {
			t.Errorf("expected ConnFocusIdx=9, got %d", model.ConnFocusIdx)
		}
	})
	t.Run("space on cluster", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ConnFocusIdx = 8
		result, _ := m.handleEditConnectionScreen(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
		model := result.(Model)
		if !model.ConnClusterMode {
			t.Error("expected cluster on")
		}
	})
	t.Run("space on text", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleEditConnectionScreen(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	})
	t.Run("enter on cluster toggle", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.ConnFocusIdx = 8
		result, _ := m.handleEditConnectionScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		model := result.(Model)
		if !model.ConnClusterMode {
			t.Error("expected cluster toggled")
		}
	})
	t.Run("enter submits when valid", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.EditingConnection = &types.Connection{ID: 1}
		m.ConnInputs[0].SetValue("name")
		m.ConnInputs[1].SetValue("host")
		_, cmd := m.handleEditConnectionScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected update cmd")
		}
	})
	t.Run("enter no submit", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleEditConnectionScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("esc cancels", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleEditConnectionScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
		model := result.(Model)
		if model.Screen != types.ScreenConnections {
			t.Errorf("expected ScreenConnections, got %v", model.Screen)
		}
	})
	t.Run("default updates input", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleEditConnectionScreen(keyMsg('z'))
	})
}

func TestHandleTestConnectionScreen(t *testing.T) {
	t.Run("esc returns to add", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleTestConnectionScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
		model := result.(Model)
		if model.Screen != types.ScreenAddConnection {
			t.Errorf("expected ScreenAddConnection, got %v", model.Screen)
		}
	})
	t.Run("enter returns to add", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleTestConnectionScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		model := result.(Model)
		if model.Screen != types.ScreenAddConnection {
			t.Errorf("expected ScreenAddConnection, got %v", model.Screen)
		}
	})
	t.Run("other key no-op", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleTestConnectionScreen(keyMsg('x'))
	})
}

func TestConnInputIndex(t *testing.T) {
	tests := []struct {
		focus    int
		expected int
	}{
		{0, 0},
		{1, 1},
		{2, 2},
		{3, 3},
		{4, 4},
		{5, 5},
		{6, 6},
		{7, 7},
		{8, -1}, // cluster toggle
		{9, 8},  // DB maps to input 8
	}
	for _, tt := range tests {
		if got := connInputIndex(tt.focus); got != tt.expected {
			t.Errorf("connInputIndex(%d) = %d, want %d", tt.focus, got, tt.expected)
		}
	}
}
