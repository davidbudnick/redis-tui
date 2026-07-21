package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/davidbudnick/redis-tui/internal/types"
)

func seedKeys(m *Model, n int) {
	m.Keys = nil
	for i := range n {
		m.Keys = append(m.Keys, types.RedisKey{Key: string(rune('a' + i)), Type: types.KeyTypeString})
	}
}

func TestHandleKeysScreen_Navigation(t *testing.T) {
	t.Run("down advances", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		seedKeys(&m, 3)
		result, cmd := m.handleKeysScreen(keyMsg('j'))
		model := result.(Model)
		if model.SelectedKeyIdx != 1 {
			t.Errorf("expected 1, got %d", model.SelectedKeyIdx)
		}
		if cmd == nil {
			t.Error("expected preview cmd")
		}
	})
	t.Run("down at bottom", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		seedKeys(&m, 3)
		m.SelectedKeyIdx = 2
		result, _ := m.handleKeysScreen(tea.KeyPressMsg{Code: tea.KeyDown})
		model := result.(Model)
		if model.SelectedKeyIdx != 2 {
			t.Errorf("expected 2, got %d", model.SelectedKeyIdx)
		}
	})
	t.Run("up decrements", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		seedKeys(&m, 3)
		m.SelectedKeyIdx = 2
		result, cmd := m.handleKeysScreen(keyMsg('k'))
		model := result.(Model)
		if model.SelectedKeyIdx != 1 {
			t.Errorf("expected 1, got %d", model.SelectedKeyIdx)
		}
		if cmd == nil {
			t.Error("expected preview cmd")
		}
	})
	t.Run("up at top", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		seedKeys(&m, 3)
		_, _ = m.handleKeysScreen(tea.KeyPressMsg{Code: tea.KeyUp})
	})
	t.Run("pgup", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		seedKeys(&m, 20)
		m.SelectedKeyIdx = 15
		result, _ := m.handleKeysScreen(tea.KeyPressMsg{Code: tea.KeyPgUp})
		model := result.(Model)
		if model.SelectedKeyIdx != 5 {
			t.Errorf("expected 5, got %d", model.SelectedKeyIdx)
		}
	})
	t.Run("pgup clamps at 0", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		seedKeys(&m, 5)
		m.SelectedKeyIdx = 3
		result, _ := m.handleKeysScreen(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
		model := result.(Model)
		if model.SelectedKeyIdx != 0 {
			t.Errorf("expected 0, got %d", model.SelectedKeyIdx)
		}
	})
	t.Run("pgdown", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		seedKeys(&m, 20)
		result, _ := m.handleKeysScreen(tea.KeyPressMsg{Code: tea.KeyPgDown})
		model := result.(Model)
		if model.SelectedKeyIdx != 10 {
			t.Errorf("expected 10, got %d", model.SelectedKeyIdx)
		}
	})
	t.Run("pgdown clamps", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		seedKeys(&m, 5)
		result, _ := m.handleKeysScreen(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
		model := result.(Model)
		if model.SelectedKeyIdx != 4 {
			t.Errorf("expected 4, got %d", model.SelectedKeyIdx)
		}
	})
	t.Run("pgdown empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleKeysScreen(tea.KeyPressMsg{Code: tea.KeyPgDown})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("home", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		seedKeys(&m, 5)
		m.SelectedKeyIdx = 3
		result, _ := m.handleKeysScreen(tea.KeyPressMsg{Code: tea.KeyHome})
		model := result.(Model)
		if model.SelectedKeyIdx != 0 {
			t.Errorf("expected 0, got %d", model.SelectedKeyIdx)
		}
	})
	t.Run("g home", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		seedKeys(&m, 5)
		m.SelectedKeyIdx = 3
		_, _ = m.handleKeysScreen(keyMsg('g'))
	})
	t.Run("home empty no-cmd", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleKeysScreen(tea.KeyPressMsg{Code: tea.KeyHome})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("end", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		seedKeys(&m, 5)
		result, _ := m.handleKeysScreen(tea.KeyPressMsg{Code: tea.KeyEnd})
		model := result.(Model)
		if model.SelectedKeyIdx != 4 {
			t.Errorf("expected 4, got %d", model.SelectedKeyIdx)
		}
	})
	t.Run("G end", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		seedKeys(&m, 5)
		_, _ = m.handleKeysScreen(keyMsg('G'))
	})
	t.Run("end empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleKeysScreen(tea.KeyPressMsg{Code: tea.KeyEnd})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
}

func TestHandleKeysScreen_Actions(t *testing.T) {
	t.Run("enter opens detail", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		seedKeys(&m, 3)
		_, cmd := m.handleKeysScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected load value cmd")
		}
	})
	t.Run("enter empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleKeysScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("y copies selected key name", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		seedKeys(&m, 2)
		_, cmd := m.handleKeysScreen(keyMsg('y'))
		if cmd == nil {
			t.Error("expected clipboard cmd")
		}
	})
	t.Run("y with no keys", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleKeysScreen(keyMsg('y'))
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("a adds key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleKeysScreen(keyMsg('a'))
		model := result.(Model)
		if model.Screen != types.ScreenAddKey {
			t.Errorf("expected ScreenAddKey, got %v", model.Screen)
		}
	})
	t.Run("n adds key", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleKeysScreen(keyMsg('n'))
		model := result.(Model)
		if model.Screen != types.ScreenAddKey {
			t.Errorf("expected ScreenAddKey, got %v", model.Screen)
		}
	})
	t.Run("d delete confirms", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		seedKeys(&m, 2)
		result, _ := m.handleKeysScreen(keyMsg('d'))
		model := result.(Model)
		if model.Screen != types.ScreenConfirmDelete {
			t.Errorf("expected ScreenConfirmDelete, got %v", model.Screen)
		}
	})
	t.Run("r reloads", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleKeysScreen(keyMsg('r'))
		if cmd == nil {
			t.Error("expected reload cmd")
		}
	})
	t.Run("l load more when cursor > 0", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.KeyCursor = 100
		_, cmd := m.handleKeysScreen(keyMsg('l'))
		if cmd == nil {
			t.Error("expected load cmd")
		}
	})
	t.Run("l no-op when cursor 0", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleKeysScreen(keyMsg('l'))
		if cmd != nil {
			t.Error("expected nil cmd")
		}
	})
	t.Run("i server info", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleKeysScreen(keyMsg('i'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("slash focuses pattern", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleKeysScreen(keyMsg('/'))
		model := result.(Model)
		if !model.Inputs.PatternInput.Focused() {
			t.Error("expected pattern focused")
		}
	})
	t.Run("f flushdb confirm", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleKeysScreen(keyMsg('f'))
		model := result.(Model)
		if model.ConfirmType != "flushdb" {
			t.Errorf("expected flushdb, got %q", model.ConfirmType)
		}
	})
	t.Run("s sorts", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		seedKeys(&m, 3)
		before := m.SortBy
		_, _ = m.handleKeysScreen(keyMsg('s'))
		_ = before
	})
	t.Run("S toggles sort", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		seedKeys(&m, 3)
		before := m.SortAsc
		result, _ := m.handleKeysScreen(keyMsg('S'))
		model := result.(Model)
		if model.SortAsc == before {
			t.Error("expected SortAsc toggled")
		}
	})
	t.Run("v search values", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleKeysScreen(keyMsg('v'))
		model := result.(Model)
		if model.Screen != types.ScreenSearchValues {
			t.Errorf("expected ScreenSearchValues, got %v", model.Screen)
		}
	})
	t.Run("e export", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleKeysScreen(keyMsg('e'))
		model := result.(Model)
		if model.Screen != types.ScreenExport {
			t.Errorf("expected ScreenExport, got %v", model.Screen)
		}
	})
	t.Run("I import", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleKeysScreen(keyMsg('I'))
		model := result.(Model)
		if model.Screen != types.ScreenImport {
			t.Errorf("expected ScreenImport, got %v", model.Screen)
		}
	})
	t.Run("p pubsub", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleKeysScreen(keyMsg('p'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("L slow log", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleKeysScreen(keyMsg('L'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("E lua", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleKeysScreen(keyMsg('E'))
		model := result.(Model)
		if model.Screen != types.ScreenLuaScript {
			t.Errorf("expected ScreenLuaScript, got %v", model.Screen)
		}
	})
	t.Run("D switch db", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleKeysScreen(keyMsg('D'))
		model := result.(Model)
		if model.Screen != types.ScreenSwitchDB {
			t.Errorf("expected ScreenSwitchDB, got %v", model.Screen)
		}
	})
	t.Run("O logs", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleKeysScreen(keyMsg('O'))
		model := result.(Model)
		if model.Screen != types.ScreenLogs {
			t.Errorf("expected ScreenLogs, got %v", model.Screen)
		}
	})
	t.Run("B bulk delete", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleKeysScreen(keyMsg('B'))
		model := result.(Model)
		if model.Screen != types.ScreenBulkDelete {
			t.Errorf("expected ScreenBulkDelete, got %v", model.Screen)
		}
	})
	t.Run("T batch ttl", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleKeysScreen(keyMsg('T'))
		model := result.(Model)
		if model.Screen != types.ScreenBatchTTL {
			t.Errorf("expected ScreenBatchTTL, got %v", model.Screen)
		}
	})
	t.Run("F favorites with conn", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentConn = &types.Connection{ID: 1}
		_, cmd := m.handleKeysScreen(keyMsg('F'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("F favorites nil conn", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleKeysScreen(keyMsg('F'))
		if cmd == nil {
			t.Error("expected cmd even without conn")
		}
	})
	t.Run("ctrl+r regex", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleKeysScreen(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
		model := result.(Model)
		if model.Screen != types.ScreenRegexSearch {
			t.Errorf("expected ScreenRegexSearch, got %v", model.Screen)
		}
	})
	t.Run("ctrl+f fuzzy", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleKeysScreen(tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
		model := result.(Model)
		if model.Screen != types.ScreenFuzzySearch {
			t.Errorf("expected ScreenFuzzySearch, got %v", model.Screen)
		}
	})
	t.Run("ctrl+l client list", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleKeysScreen(tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("m live metrics", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleKeysScreen(keyMsg('m'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("M mem stats", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleKeysScreen(keyMsg('M'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("C cluster info", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleKeysScreen(keyMsg('C'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("K compare", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleKeysScreen(keyMsg('K'))
		model := result.(Model)
		if model.Screen != types.ScreenCompareKeys {
			t.Errorf("expected ScreenCompareKeys, got %v", model.Screen)
		}
	})
	t.Run("P templates", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleKeysScreen(keyMsg('P'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("ctrl+h recent", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.CurrentConn = &types.Connection{ID: 1}
		_, cmd := m.handleKeysScreen(tea.KeyPressMsg{Code: 'h', Mod: tea.ModCtrl})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("ctrl+h recent nil conn", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleKeysScreen(tea.KeyPressMsg{Code: 'h', Mod: tea.ModCtrl})
	})
	t.Run("ctrl+e toggle keyspace on", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, cmd := m.handleKeysScreen(tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl})
		model := result.(Model)
		if !model.KeyspaceSubActive {
			t.Error("expected KeyspaceSubActive")
		}
		if cmd == nil {
			t.Error("expected subscribe cmd")
		}
	})
	t.Run("ctrl+e toggle keyspace with sendfunc", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		fn := func(tea.Msg) {}
		m.SendFunc = &fn
		_, _ = m.handleKeysScreen(tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl})
	})
	t.Run("ctrl+e toggle keyspace off", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.KeyspaceSubActive = true
		result, _ := m.handleKeysScreen(tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl})
		model := result.(Model)
		if model.KeyspaceSubActive {
			t.Error("expected KeyspaceSubActive off")
		}
	})
	t.Run("W tree view", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleKeysScreen(keyMsg('W'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("ctrl+g redis config", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, cmd := m.handleKeysScreen(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("ctrl+x expiring keys", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{
			{Key: "a", TTL: 30 * time.Second},
			{Key: "b", TTL: 500 * time.Second},
			{Key: "c", TTL: 0},
		}
		result, _ := m.handleKeysScreen(tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl})
		model := result.(Model)
		if len(model.ExpiringKeys) != 1 {
			t.Errorf("expected 1 expiring, got %d", len(model.ExpiringKeys))
		}
	})
	t.Run("esc clear pattern", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.KeyPattern = "foo"
		_, cmd := m.handleKeysScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
		if cmd == nil {
			t.Error("expected load cmd")
		}
	})
	t.Run("esc exit", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		result, _ := m.handleKeysScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
		model := result.(Model)
		if model.Screen != types.ScreenConnections {
			t.Errorf("expected ScreenConnections, got %v", model.Screen)
		}
	})
}

func TestHandleKeysScreen_PatternInput(t *testing.T) {
	t.Run("enter wraps pattern", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Inputs.PatternInput.Focus()
		m.Inputs.PatternInput.SetValue("user")
		result, _ := m.handleKeysScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		model := result.(Model)
		if model.KeyPattern != "*user*" {
			t.Errorf("expected *user*, got %q", model.KeyPattern)
		}
	})
	t.Run("enter keeps wildcard pattern", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Inputs.PatternInput.Focus()
		m.Inputs.PatternInput.SetValue("user:*")
		result, _ := m.handleKeysScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		model := result.(Model)
		if model.KeyPattern != "user:*" {
			t.Errorf("expected user:*, got %q", model.KeyPattern)
		}
	})
	t.Run("enter empty", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Inputs.PatternInput.Focus()
		_, cmd := m.handleKeysScreen(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Error("expected reload cmd")
		}
	})
	t.Run("esc clears", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Inputs.PatternInput.Focus()
		m.Inputs.PatternInput.SetValue("foo")
		_, cmd := m.handleKeysScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
		if cmd == nil {
			t.Error("expected reload cmd")
		}
	})
	t.Run("default debounces", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Inputs.PatternInput.Focus()
		_, cmd := m.handleKeysScreen(keyMsg('x'))
		if cmd == nil {
			t.Error("expected debounce cmd")
		}
	})
}

func TestSortKeys(t *testing.T) {
	t.Run("name to type", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{
			{Key: "b", Type: types.KeyTypeList, TTL: 2},
			{Key: "a", Type: types.KeyTypeString, TTL: 1},
		}
		m.SortBy = "name"
		m.SortAsc = true
		m.sortKeys()
		if m.SortBy != "type" {
			t.Errorf("expected SortBy=type, got %q", m.SortBy)
		}
	})
	t.Run("type to ttl", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "a", TTL: 2}, {Key: "b", TTL: 1}}
		m.SortBy = "type"
		m.sortKeys()
		if m.SortBy != "ttl" {
			t.Errorf("expected SortBy=ttl, got %q", m.SortBy)
		}
	})
	t.Run("ttl to name", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "b"}, {Key: "a"}}
		m.SortBy = "ttl"
		m.sortKeys()
		if m.SortBy != "name" {
			t.Errorf("expected SortBy=name, got %q", m.SortBy)
		}
	})
	t.Run("default to name", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "b"}, {Key: "a"}}
		m.SortBy = "unknown"
		m.sortKeys()
		if m.SortBy != "name" {
			t.Errorf("expected SortBy=name, got %q", m.SortBy)
		}
	})
	t.Run("descending", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		m.Keys = []types.RedisKey{{Key: "a"}, {Key: "b"}}
		m.SortBy = "ttl" // will become name
		m.SortAsc = false
		m.sortKeys()
		if m.Keys[0].Key != "b" {
			t.Errorf("expected b first, got %q", m.Keys[0].Key)
		}
	})
}

func TestHandleKeyDetailScreen(t *testing.T) {
	newModelWithKey := func(t *testing.T, keyType types.KeyType) (Model, *types.RedisKey) {
		m, _, _ := newTestModel(t)
		key := &types.RedisKey{Key: "foo", Type: keyType}
		m.CurrentKey = key
		m.CurrentValue = types.RedisValue{
			Type:        keyType,
			StringValue: "hello",
			JSONValue:   `{"a":1}`,
			ListValue:   []string{"a", "b"},
			SetValue:    []string{"x", "y"},
			HashValue:   map[string]string{"k": "v"},
		}
		return m, key
	}

	t.Run("d delete", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		result, _ := m.handleKeyDetailScreen(keyMsg('d'))
		model := result.(Model)
		if model.Screen != types.ScreenConfirmDelete {
			t.Errorf("expected ScreenConfirmDelete, got %v", model.Screen)
		}
	})
	t.Run("d delete nil", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleKeyDetailScreen(keyMsg('d'))
	})
	t.Run("t ttl editor", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		result, _ := m.handleKeyDetailScreen(keyMsg('t'))
		model := result.(Model)
		if model.Screen != types.ScreenTTLEditor {
			t.Errorf("expected ScreenTTLEditor, got %v", model.Screen)
		}
	})
	t.Run("t ttl nil", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleKeyDetailScreen(keyMsg('t'))
	})
	t.Run("r reload", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		_, cmd := m.handleKeyDetailScreen(keyMsg('r'))
		if cmd == nil {
			t.Error("expected reload cmd")
		}
	})
	t.Run("r reload nil", func(t *testing.T) {
		m, _, _ := newTestModel(t)
		_, _ = m.handleKeyDetailScreen(keyMsg('r'))
	})
	t.Run("e edit string", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		result, _ := m.handleKeyDetailScreen(keyMsg('e'))
		model := result.(Model)
		if model.Screen != types.ScreenEditValue {
			t.Errorf("expected ScreenEditValue, got %v", model.Screen)
		}
	})
	t.Run("e edit json", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeJSON)
		result, _ := m.handleKeyDetailScreen(keyMsg('e'))
		model := result.(Model)
		if model.Screen != types.ScreenEditValue {
			t.Errorf("expected ScreenEditValue, got %v", model.Screen)
		}
	})
	t.Run("e edit string with json content", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		m.CurrentValue.StringValue = `{"a":1}`
		_, _ = m.handleKeyDetailScreen(keyMsg('e'))
	})
	t.Run("e edit non-string non-json", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeList)
		_, _ = m.handleKeyDetailScreen(keyMsg('e'))
	})
	t.Run("a add to collection", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeList)
		result, _ := m.handleKeyDetailScreen(keyMsg('a'))
		model := result.(Model)
		if model.Screen != types.ScreenAddToCollection {
			t.Errorf("expected ScreenAddToCollection, got %v", model.Screen)
		}
	})
	t.Run("a string no-op", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		_, _ = m.handleKeyDetailScreen(keyMsg('a'))
	})
	t.Run("x remove from collection", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeList)
		result, _ := m.handleKeyDetailScreen(keyMsg('x'))
		model := result.(Model)
		if model.Screen != types.ScreenRemoveFromCollection {
			t.Errorf("expected ScreenRemoveFromCollection, got %v", model.Screen)
		}
	})
	t.Run("x string no-op", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		_, _ = m.handleKeyDetailScreen(keyMsg('x'))
	})
	t.Run("x hll no-op", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeHyperLogLog)
		_, _ = m.handleKeyDetailScreen(keyMsg('x'))
	})
	t.Run("R rename", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		result, _ := m.handleKeyDetailScreen(keyMsg('R'))
		model := result.(Model)
		if model.Screen != types.ScreenRenameKey {
			t.Errorf("expected ScreenRenameKey, got %v", model.Screen)
		}
	})
	t.Run("c copy", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		result, _ := m.handleKeyDetailScreen(keyMsg('c'))
		model := result.(Model)
		if model.Screen != types.ScreenCopyKey {
			t.Errorf("expected ScreenCopyKey, got %v", model.Screen)
		}
	})
	t.Run("f favorite add", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		m.CurrentConn = &types.Connection{ID: 1, Name: "n"}
		_, cmd := m.handleKeyDetailScreen(keyMsg('f'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("f favorite remove", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		m.CurrentConn = &types.Connection{ID: 1, Name: "n"}
		m.CurrentKey.IsFavorite = true
		_, cmd := m.handleKeyDetailScreen(keyMsg('f'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("w watch start", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		_, cmd := m.handleKeyDetailScreen(keyMsg('w'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("w watch stop", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		m.WatchActive = true
		m.WatchKey = "foo"
		result, _ := m.handleKeyDetailScreen(keyMsg('w'))
		model := result.(Model)
		if model.WatchActive {
			t.Error("expected watch stopped")
		}
	})
	t.Run("h history", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		_, cmd := m.handleKeyDetailScreen(keyMsg('h'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("y copy clipboard", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		_, cmd := m.handleKeyDetailScreen(keyMsg('y'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("y copy protobuf decoded", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeProtobuf)
		m.CurrentValue = types.RedisValue{
			Type:         types.KeyTypeProtobuf,
			StringValue:  "raw-binary",
			DecodedValue: "1: \"decoded\"\n",
		}
		_, cmd := m.handleKeyDetailScreen(keyMsg('y'))
		if cmd == nil {
			t.Error("expected cmd")
		}
	})
	t.Run("a ignored for protobuf", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeProtobuf)
		result, _ := m.handleKeyDetailScreen(keyMsg('a'))
		model := result.(Model)
		if model.Screen == types.ScreenAddToCollection {
			t.Error("protobuf should not open add-to-collection")
		}
	})
	t.Run("x ignored for protobuf", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeProtobuf)
		result, _ := m.handleKeyDetailScreen(keyMsg('x'))
		model := result.(Model)
		if model.Screen == types.ScreenRemoveFromCollection {
			t.Error("protobuf should not open remove-from-collection")
		}
	})
	t.Run("J json path", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		result, _ := m.handleKeyDetailScreen(keyMsg('J'))
		model := result.(Model)
		if model.Screen != types.ScreenJSONPath {
			t.Errorf("expected ScreenJSONPath, got %v", model.Screen)
		}
	})
	t.Run("J non-string no-op", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeList)
		_, _ = m.handleKeyDetailScreen(keyMsg('J'))
	})
	t.Run("up cursor detail", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		m.Height = 40
		m.Width = 120
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeString, StringValue: strings.Repeat("line\n", 20)}
		m.DetailCursor = 3
		result, _ := m.handleKeyDetailScreen(tea.KeyPressMsg{Code: tea.KeyUp})
		model := result.(Model)
		if model.DetailCursor != 2 {
			t.Errorf("expected cursor 2, got %d", model.DetailCursor)
		}
	})
	t.Run("up scroll item", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeList)
		m.SelectedItemIdx = 1
		m.DetailCursor = 0
		result, _ := m.handleKeyDetailScreen(keyMsg('k'))
		model := result.(Model)
		if model.SelectedItemIdx != 0 {
			t.Errorf("expected 0, got %d", model.SelectedItemIdx)
		}
	})
	t.Run("down cursor", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeString, StringValue: "a\nb\nc\nd\ne"}
		m.DetailCursor = 0
		result, _ := m.handleKeyDetailScreen(keyMsg('j'))
		if result.(Model).DetailCursor != 1 {
			t.Errorf("expected cursor 1, got %d", result.(Model).DetailCursor)
		}
	})
	t.Run("down cursor clamps at end", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeString, StringValue: "only"}
		m.DetailCursor = 0
		result, _ := m.handleKeyDetailScreen(keyMsg('j'))
		if result.(Model).DetailCursor != 0 {
			t.Errorf("expected cursor stay 0, got %d", result.(Model).DetailCursor)
		}
	})
	t.Run("pgdown clamps at end", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeString, StringValue: "a\nb"}
		m.DetailCursor = 0
		result, _ := m.handleKeyDetailScreen(tea.KeyPressMsg{Code: tea.KeyPgDown})
		if result.(Model).DetailCursor != 1 {
			t.Errorf("expected cursor 1, got %d", result.(Model).DetailCursor)
		}
	})
	t.Run("end empty content", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeString, StringValue: ""}
		result, _ := m.handleKeyDetailScreen(tea.KeyPressMsg{Code: tea.KeyEnd})
		if result.(Model).DetailCursor != 0 {
			t.Errorf("expected 0, got %d", result.(Model).DetailCursor)
		}
	})
	t.Run("pgup detail", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		m.Height = 40
		m.Width = 120
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeString, StringValue: strings.Repeat("line\n", 40)}
		m.DetailCursor = 20
		result, _ := m.handleKeyDetailScreen(tea.KeyPressMsg{Code: tea.KeyPgUp})
		if result.(Model).DetailCursor != 10 {
			t.Errorf("expected cursor 10, got %d", result.(Model).DetailCursor)
		}
	})
	t.Run("pgdown detail", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		m.CurrentValue = types.RedisValue{Type: types.KeyTypeString, StringValue: strings.Repeat("line\n", 50)}
		_, _ = m.handleKeyDetailScreen(tea.KeyPressMsg{Code: tea.KeyPgDown})
	})
	t.Run("home detail", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		m.DetailCursor = 5
		result, _ := m.handleKeyDetailScreen(tea.KeyPressMsg{Code: tea.KeyHome})
		model := result.(Model)
		if model.DetailCursor != 0 {
			t.Errorf("expected cursor 0, got %d", model.DetailCursor)
		}
	})
	t.Run("g home", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		_, _ = m.handleKeyDetailScreen(keyMsg('g'))
	})
	t.Run("end detail", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		_, _ = m.handleKeyDetailScreen(tea.KeyPressMsg{Code: tea.KeyEnd})
	})
	t.Run("G end", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		_, _ = m.handleKeyDetailScreen(keyMsg('G'))
	})
	t.Run("esc exit", func(t *testing.T) {
		m, _ := newModelWithKey(t, types.KeyTypeString)
		result, _ := m.handleKeyDetailScreen(tea.KeyPressMsg{Code: tea.KeyEsc})
		model := result.(Model)
		if model.Screen != types.ScreenKeys {
			t.Errorf("expected ScreenKeys, got %v", model.Screen)
		}
		if model.CurrentKey != nil {
			t.Error("expected CurrentKey cleared")
		}
	})
}

func TestCreateVimEditor(t *testing.T) {
	ed := createVimEditor("hello", 80, 24, "f.txt")
	if ed == nil {
		t.Fatal("expected non-nil editor")
	}
	if ed.Value() != "hello" {
		t.Errorf("expected content hello, got %q", ed.Value())
	}
	if ed.FileName() != "f.txt" {
		t.Errorf("expected fileName f.txt, got %q", ed.FileName())
	}
	edNoFile := createVimEditor("content", 80, 24, "")
	if edNoFile == nil {
		t.Fatal("expected non-nil editor")
	}
	if edNoFile.FileName() != "" {
		t.Errorf("expected empty fileName, got %q", edNoFile.FileName())
	}
}

func TestGetCollectionLength(t *testing.T) {
	tests := []struct {
		name     string
		value    types.RedisValue
		expected int
	}{
		{"list", types.RedisValue{Type: types.KeyTypeList, ListValue: []string{"a", "b"}}, 2},
		{"set", types.RedisValue{Type: types.KeyTypeSet, SetValue: []string{"a"}}, 1},
		{"zset", types.RedisValue{Type: types.KeyTypeZSet, ZSetValue: []types.ZSetMember{{Member: "a"}}}, 1},
		{"hash", types.RedisValue{Type: types.KeyTypeHash, HashValue: map[string]string{"k": "v"}}, 1},
		{"stream", types.RedisValue{Type: types.KeyTypeStream, StreamValue: []types.StreamEntry{{ID: "1"}}}, 1},
		{"geo", types.RedisValue{Type: types.KeyTypeGeo, GeoValue: []types.GeoMember{{Name: "a"}}}, 1},
		{"hll", types.RedisValue{Type: types.KeyTypeHyperLogLog}, 0},
		{"json", types.RedisValue{Type: types.KeyTypeJSON}, 0},
		{"bitmap", types.RedisValue{Type: types.KeyTypeBitmap}, 0},
		{"unknown", types.RedisValue{Type: "unknown"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _, _ := newTestModel(t)
			m.CurrentValue = tt.value
			if got := m.getCollectionLength(); got != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, got)
			}
		})
	}
}
