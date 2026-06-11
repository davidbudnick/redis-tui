package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/davidbudnick/redis-tui/internal/types"
)

func ctrlY() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl}
}

func TestFocusedInputValue_SingleInputScreens(t *testing.T) {
	cases := []struct {
		name   string
		screen types.Screen
		focus  func(m *Model)
	}{
		{"ttl editor", types.ScreenTTLEditor, func(m *Model) { m.Inputs.TTLInput.Focus(); m.Inputs.TTLInput.SetValue("60") }},
		{"rename key", types.ScreenRenameKey, func(m *Model) { m.Inputs.RenameInput.Focus(); m.Inputs.RenameInput.SetValue("60") }},
		{"copy key", types.ScreenCopyKey, func(m *Model) { m.Inputs.CopyInput.Focus(); m.Inputs.CopyInput.SetValue("60") }},
		{"switch db", types.ScreenSwitchDB, func(m *Model) { m.Inputs.DBSwitchInput.Focus(); m.Inputs.DBSwitchInput.SetValue("60") }},
		{"search values", types.ScreenSearchValues, func(m *Model) { m.Inputs.SearchValueInput.Focus(); m.Inputs.SearchValueInput.SetValue("60") }},
		{"export", types.ScreenExport, func(m *Model) { m.Inputs.ExportInput.Focus(); m.Inputs.ExportInput.SetValue("60") }},
		{"import", types.ScreenImport, func(m *Model) { m.Inputs.ImportInput.Focus(); m.Inputs.ImportInput.SetValue("60") }},
		{"lua script", types.ScreenLuaScript, func(m *Model) { m.Inputs.LuaScriptInput.Focus(); m.Inputs.LuaScriptInput.SetValue("60") }},
		{"bulk delete", types.ScreenBulkDelete, func(m *Model) { m.Inputs.BulkDeleteInput.Focus(); m.Inputs.BulkDeleteInput.SetValue("60") }},
		{"batch ttl", types.ScreenBatchTTL, func(m *Model) { m.Inputs.BatchTTLPattern.Focus(); m.Inputs.BatchTTLPattern.SetValue("60") }},
		{"regex search", types.ScreenRegexSearch, func(m *Model) { m.Inputs.RegexSearchInput.Focus(); m.Inputs.RegexSearchInput.SetValue("60") }},
		{"fuzzy search", types.ScreenFuzzySearch, func(m *Model) { m.Inputs.FuzzySearchInput.Focus(); m.Inputs.FuzzySearchInput.SetValue("60") }},
		{"compare keys", types.ScreenCompareKeys, func(m *Model) { m.Inputs.CompareKey2Input.Focus(); m.Inputs.CompareKey2Input.SetValue("60") }},
		{"json path", types.ScreenJSONPath, func(m *Model) { m.Inputs.JSONPathInput.Focus(); m.Inputs.JSONPathInput.SetValue("60") }},
		{"redis config", types.ScreenRedisConfig, func(m *Model) { m.Inputs.ConfigEditInput.Focus(); m.Inputs.ConfigEditInput.SetValue("60") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, _, _ := newTestModel(t)
			m.Screen = tc.screen
			tc.focus(&m)
			value, ok := m.focusedInputValue()
			if !ok || value != "60" {
				t.Errorf("expected (60, true), got (%q, %v)", value, ok)
			}
		})
	}
}

func TestFocusedInputValue_SliceInputScreens(t *testing.T) {
	t.Run("connection screens", func(t *testing.T) {
		for _, screen := range []types.Screen{types.ScreenAddConnection, types.ScreenEditConnection} {
			m, _, _ := newTestModel(t)
			m.Screen = screen
			m.ConnInputs[0].SetValue("prod")
			value, ok := m.focusedInputValue()
			if !ok || value != "prod" {
				t.Errorf("expected (prod, true), got (%q, %v)", value, ok)
			}
		}
	})
	t.Run("add key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenAddKey
		m.AddKeyInputs[0].SetValue("user:1")
		value, ok := m.focusedInputValue()
		if !ok || value != "user:1" {
			t.Errorf("expected (user:1, true), got (%q, %v)", value, ok)
		}
	})
	t.Run("add to collection", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenAddToCollection
		m.AddCollectionInput[0].SetValue("member")
		value, ok := m.focusedInputValue()
		if !ok || value != "member" {
			t.Errorf("expected (member, true), got (%q, %v)", value, ok)
		}
	})
	t.Run("pubsub screens", func(t *testing.T) {
		for _, screen := range []types.Screen{types.ScreenPubSub, types.ScreenPublishMessage} {
			m, _, _ := newTestModel(t)
			m.Screen = screen
			m.PubSubInput[0].SetValue("events")
			value, ok := m.focusedInputValue()
			if !ok || value != "events" {
				t.Errorf("expected (events, true), got (%q, %v)", value, ok)
			}
		}
	})
}

func TestFocusedInputValue_KeysScreen(t *testing.T) {
	t.Run("focused pattern input", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenKeys
		m.Inputs.PatternInput.Focus()
		m.Inputs.PatternInput.SetValue("user:*")
		value, ok := m.focusedInputValue()
		if !ok || value != "user:*" {
			t.Errorf("expected (user:*, true), got (%q, %v)", value, ok)
		}
	})
	t.Run("unfocused pattern input", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenKeys
		if _, ok := m.focusedInputValue(); ok {
			t.Error("expected ok=false")
		}
	})
}

func TestFocusedInputValue_ScreenWithoutInputs(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Screen = types.ScreenHelp
	if _, ok := m.focusedInputValue(); ok {
		t.Error("expected ok=false")
	}
}

func TestFocusedValue_NoneFocused(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Inputs.BatchTTLInput.SetValue("60")
	if _, ok := focusedValue(m.Inputs.BatchTTLInput, m.Inputs.BatchTTLPattern); ok {
		t.Error("expected ok=false when nothing focused")
	}
}

func TestUpdate_CtrlY(t *testing.T) {
	t.Run("copies focused input value", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenRenameKey
		m.Inputs.RenameInput.Focus()
		m.Inputs.RenameInput.SetValue("new-name")
		_, cmd := m.Update(ctrlY())
		if cmd == nil {
			t.Error("expected clipboard cmd")
		}
	})
	t.Run("empty focused input falls through", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenRenameKey
		m.Inputs.RenameInput.Focus()
		_, cmd := m.Update(ctrlY())
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("no focused input falls through", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenHelp
		result, _ := m.Update(ctrlY())
		if result.(Model).Screen != types.ScreenHelp {
			t.Error("expected screen unchanged")
		}
	})
}
