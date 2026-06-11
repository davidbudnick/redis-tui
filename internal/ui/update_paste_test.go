package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/davidbudnick/redis-tui/internal/types"
)

func pasteMsg(content string) tea.PasteMsg {
	return tea.PasteMsg{Content: content}
}

func TestUpdate_PasteMsg_SingleInputScreens(t *testing.T) {
	cases := []struct {
		name   string
		screen types.Screen
		focus  func(m *Model)
		value  func(m Model) string
	}{
		{"ttl editor", types.ScreenTTLEditor, func(m *Model) { m.Inputs.TTLInput.Focus() }, func(m Model) string { return m.Inputs.TTLInput.Value() }},
		{"rename key", types.ScreenRenameKey, func(m *Model) { m.Inputs.RenameInput.Focus() }, func(m Model) string { return m.Inputs.RenameInput.Value() }},
		{"copy key", types.ScreenCopyKey, func(m *Model) { m.Inputs.CopyInput.Focus() }, func(m Model) string { return m.Inputs.CopyInput.Value() }},
		{"switch db", types.ScreenSwitchDB, func(m *Model) { m.Inputs.DBSwitchInput.Focus() }, func(m Model) string { return m.Inputs.DBSwitchInput.Value() }},
		{"search values", types.ScreenSearchValues, func(m *Model) { m.Inputs.SearchValueInput.Focus() }, func(m Model) string { return m.Inputs.SearchValueInput.Value() }},
		{"export", types.ScreenExport, func(m *Model) { m.Inputs.ExportInput.Focus() }, func(m Model) string { return m.Inputs.ExportInput.Value() }},
		{"import", types.ScreenImport, func(m *Model) { m.Inputs.ImportInput.Focus() }, func(m Model) string { return m.Inputs.ImportInput.Value() }},
		{"lua script", types.ScreenLuaScript, func(m *Model) { m.Inputs.LuaScriptInput.Focus() }, func(m Model) string { return m.Inputs.LuaScriptInput.Value() }},
		{"bulk delete", types.ScreenBulkDelete, func(m *Model) { m.Inputs.BulkDeleteInput.Focus() }, func(m Model) string { return m.Inputs.BulkDeleteInput.Value() }},
		{"regex search", types.ScreenRegexSearch, func(m *Model) { m.Inputs.RegexSearchInput.Focus() }, func(m Model) string { return m.Inputs.RegexSearchInput.Value() }},
		{"fuzzy search", types.ScreenFuzzySearch, func(m *Model) { m.Inputs.FuzzySearchInput.Focus() }, func(m Model) string { return m.Inputs.FuzzySearchInput.Value() }},
		{"json path", types.ScreenJSONPath, func(m *Model) { m.Inputs.JSONPathInput.Focus() }, func(m Model) string { return m.Inputs.JSONPathInput.Value() }},
		{"redis config", types.ScreenRedisConfig, func(m *Model) { m.Inputs.ConfigEditInput.Focus() }, func(m Model) string { return m.Inputs.ConfigEditInput.Value() }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, _, _ := newTestModel(t)
			m.Screen = tc.screen
			tc.focus(&m)
			result, _ := m.Update(pasteMsg("pasted"))
			if got := tc.value(result.(Model)); got != "pasted" {
				t.Errorf("expected pasted, got %q", got)
			}
		})
	}
}

func TestUpdate_PasteMsg_ConnectionScreens(t *testing.T) {
	for _, screen := range []types.Screen{types.ScreenAddConnection, types.ScreenEditConnection} {
		m, _, _ := newTestModel(t)
		m.Screen = screen
		result, _ := m.Update(pasteMsg("my-conn"))
		model := result.(Model)
		if got := model.ConnInputs[0].Value(); got != "my-conn" {
			t.Errorf("expected paste in focused name input, got %q", got)
		}
		if got := model.ConnInputs[1].Value(); got != "localhost" {
			t.Errorf("expected unfocused host input untouched, got %q", got)
		}
	}
}

func TestUpdate_PasteMsg_KeysScreenSearch(t *testing.T) {
	t.Run("focused pattern input receives paste and debounces", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenKeys
		m.Inputs.PatternInput.Focus()
		m.SearchSeq = 3
		result, cmd := m.Update(pasteMsg("user:"))
		model := result.(Model)
		if got := model.Inputs.PatternInput.Value(); got != "user:" {
			t.Errorf("expected user:, got %q", got)
		}
		if model.SearchSeq != 4 {
			t.Errorf("expected SearchSeq 4, got %d", model.SearchSeq)
		}
		if cmd == nil {
			t.Fatal("expected debounce cmd")
		}
		batch, ok := cmd().(tea.BatchMsg)
		if !ok {
			t.Fatal("expected BatchMsg")
		}
		found := false
		for _, c := range batch {
			if c == nil {
				continue
			}
			if debounce, ok := c().(types.SearchDebounceMsg); ok {
				found = true
				if debounce.Seq != 4 {
					t.Errorf("expected Seq 4, got %d", debounce.Seq)
				}
			}
		}
		if !found {
			t.Error("expected SearchDebounceMsg in batch")
		}
	})
	t.Run("unfocused pattern input ignores paste", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenKeys
		result, cmd := m.Update(pasteMsg("user:"))
		if got := result.(Model).Inputs.PatternInput.Value(); got != "" {
			t.Errorf("expected empty, got %q", got)
		}
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
}

func TestUpdate_PasteMsg_AddKeyScreen(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Screen = types.ScreenAddKey
	result, _ := m.Update(pasteMsg("key-name"))
	model := result.(Model)
	if got := model.AddKeyInputs[0].Value(); got != "key-name" {
		t.Errorf("expected key-name, got %q", got)
	}
	if got := model.AddKeyInputs[1].Value(); got != "" {
		t.Errorf("expected unfocused value input untouched, got %q", got)
	}
}

func TestUpdate_PasteMsg_AddToCollectionScreen(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Screen = types.ScreenAddToCollection
	result, _ := m.Update(pasteMsg("member"))
	if got := result.(Model).AddCollectionInput[0].Value(); got != "member" {
		t.Errorf("expected member, got %q", got)
	}
}

func TestUpdate_PasteMsg_PubSubScreens(t *testing.T) {
	for _, screen := range []types.Screen{types.ScreenPubSub, types.ScreenPublishMessage} {
		m, _, _ := newTestModel(t)
		m.Screen = screen
		result, _ := m.Update(pasteMsg("channel"))
		if got := result.(Model).PubSubInput[0].Value(); got != "channel" {
			t.Errorf("expected channel, got %q", got)
		}
	}
}

func TestUpdate_PasteMsg_EditValueScreen(t *testing.T) {
	t.Run("paste inserts into editor", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenEditValue
		m.VimEditor = createVimEditor("", 80, 24, "test.txt")
		result, _ := m.Update(pasteMsg("pasted value"))
		if got := result.(Model).VimEditor.Value(); !strings.Contains(got, "pasted value") {
			t.Errorf("expected editor to contain pasted value, got %q", got)
		}
	})
	t.Run("nil editor is a no-op", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenEditValue
		m.VimEditor = nil
		_, cmd := m.Update(pasteMsg("x"))
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
}

func TestUpdate_PasteMsg_BatchTTLScreen(t *testing.T) {
	t.Run("ttl input focused", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenBatchTTL
		m.Inputs.BatchTTLInput.Focus()
		result, _ := m.Update(pasteMsg("3600"))
		model := result.(Model)
		if got := model.Inputs.BatchTTLInput.Value(); got != "3600" {
			t.Errorf("expected 3600, got %q", got)
		}
		if got := model.Inputs.BatchTTLPattern.Value(); got != "" {
			t.Errorf("expected pattern input untouched, got %q", got)
		}
	})
	t.Run("pattern input focused", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Screen = types.ScreenBatchTTL
		m.Inputs.BatchTTLPattern.Focus()
		result, _ := m.Update(pasteMsg("user:*"))
		model := result.(Model)
		if got := model.Inputs.BatchTTLPattern.Value(); got != "user:*" {
			t.Errorf("expected user:*, got %q", got)
		}
		if got := model.Inputs.BatchTTLInput.Value(); got != "" {
			t.Errorf("expected ttl input untouched, got %q", got)
		}
	})
}

func TestUpdate_PasteMsg_CompareKeysScreen(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Screen = types.ScreenCompareKeys
	m.Inputs.CompareKey2Input.Focus()
	result, _ := m.Update(pasteMsg("key2"))
	model := result.(Model)
	if got := model.Inputs.CompareKey2Input.Value(); got != "key2" {
		t.Errorf("expected key2, got %q", got)
	}
	if got := model.Inputs.CompareKey1Input.Value(); got != "" {
		t.Errorf("expected key1 input untouched, got %q", got)
	}
}

func TestUpdate_PasteMsg_ScreenWithoutInputs(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.Screen = types.ScreenConnections
	_, cmd := m.Update(pasteMsg("ignored"))
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}
